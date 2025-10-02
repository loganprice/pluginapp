package manager

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/example/grpc-plugin-app/pkg/grpc"
	"github.com/example/grpc-plugin-app/pkg/plugin"
)

// PluginManager handles plugin lifecycle management
type PluginManager struct {
	config     *AppConfig
	plugins    map[string]*ManagedPlugin
	mu         sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// ManagedPlugin represents a managed plugin instance
type ManagedPlugin struct {
	Name       string
	Config     plugin.PluginConfig
	Client     plugin.Plugin
	GRPCClient *grpc.Client
	Cmd        *exec.Cmd
	RestartCnt int
	LastError  error
	Params     map[string]string
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(config *AppConfig) *PluginManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &PluginManager{
		config:     config,
		plugins:    make(map[string]*ManagedPlugin),
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// StartPlugin starts a plugin and manages its lifecycle
func (pm *PluginManager) StartPlugin(name string, pluginConfig plugin.PluginConfig, params map[string]string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s is already running", name)
	}

	config := pluginConfig

	var client plugin.Plugin
	var clientErr error
	var process *exec.Cmd

	if config.Type == plugin.PluginTypeRemote {
		// For remote plugins, just connect, don't start a process
		client, clientErr = grpc.NewClientWithAddress(config.Address)
	} else {
		// For local plugins, start the process and then connect
		cmd, args, err := config.GetStartCommand(config.Port, params)
		if err != nil {
			return fmt.Errorf("failed to get start command: %v", err)
		}

		process = exec.CommandContext(pm.ctx, cmd, args...)
		process.Dir = config.WorkingDir
		process.Stderr = os.Stderr
		process.Stdout = os.Stdout

		process.Env = os.Environ()
		for k, v := range config.Environment {
			process.Env = append(process.Env, fmt.Sprintf("%s=%s", k, v))
		}

		if err := process.Start(); err != nil {
			return fmt.Errorf("failed to start plugin %s: %v", name, err)
		}

		// Wait for the plugin to start and be ready
		for retries := 0; retries < 5; retries++ {
			time.Sleep(time.Second)
			client, clientErr = grpc.NewClient(config.Port)
			if clientErr == nil {
				break
			}
		}
	}

	if clientErr != nil {
		if process != nil {
			process.Process.Kill()
		}
		return fmt.Errorf("failed to connect to plugin %s: %v", name, clientErr)
	}

	grpcClient, ok := client.(*grpc.Client)
	if !ok {
		if process != nil {
			process.Process.Kill()
		}
		return fmt.Errorf("invalid client type for plugin %s", name)
	}

	grpcClient.Name = name

	managed := &ManagedPlugin{
		Name:       name,
		Config:     config,
		Client:     client,
		GRPCClient: grpcClient,
		Cmd:        process, // Cmd will be nil for remote plugins
		Params:     params,
	}

	// For local plugins, enable health checking with automatic restart
	if managed.Cmd != nil {
		pm.EnableHealthCheck(managed)
	}

	pm.plugins[name] = managed
	return nil
}

// EnableHealthCheck configures and starts the health monitor for a local plugin
func (pm *PluginManager) EnableHealthCheck(plug *ManagedPlugin) {
	config := HealthCheck{
		Interval:   time.Second * 30,
		MaxRetries: 3,
		RetryDelay: time.Second * 5,
		OnUnhealthy: func(err error) {
			pm.mu.Lock()
			defer pm.mu.Unlock()

			plug.LastError = err
			if plug.RestartCnt < 3 {
				plug.RestartCnt++
				pm.restartPlugin(plug)
			}
		},
	}
	go MonitorPluginHealth(pm.ctx, plug.GRPCClient, config)
}

// StopPlugin stops a running plugin
func (pm *PluginManager) StopPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s is not running", name)
	}

	if err := plugin.Client.Close(); err != nil {
		log.Printf("Warning: failed to close plugin client for %s: %v", name, err)
	}

	// Only try to kill the process if it's a local plugin
	if plugin.Cmd != nil && plugin.Cmd.Process != nil {
		if err := plugin.Cmd.Process.Kill(); err != nil {
			log.Printf("Warning: failed to kill plugin process for %s: %v", name, err)
		}
	}

	delete(pm.plugins, name)
	return nil
}

// StopAll stops all running plugins
func (pm *PluginManager) StopAll() {
	pm.cancelFunc()
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, plugin := range pm.plugins {
		plugin.Client.Close()
		// Only try to kill the process if it's a local plugin
		if plugin.Cmd != nil && plugin.Cmd.Process != nil {
			plugin.Cmd.Process.Kill()
		}
		delete(pm.plugins, name)
	}
}

// GetPlugin returns a plugin client by name
func (pm *PluginManager) GetPlugin(name string) (plugin.Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s is not running", name)
	}

	return plugin.Client, nil
}

// restartPlugin attempts to restart a failed plugin
func (pm *PluginManager) restartPlugin(plug *ManagedPlugin) {
	if plug.Cmd == nil {
		plug.LastError = fmt.Errorf("cannot restart a non-local plugin")
		return
	}

	plug.Client.Close()
	plug.Cmd.Process.Kill()

	// Get the appropriate start command based on plugin type
	cmd, args, err := plug.Config.GetStartCommand(plug.Config.Port, plug.Params)
	if err != nil {
		plug.LastError = fmt.Errorf("failed to get restart command: %v", err)
		return
	}

	process := exec.CommandContext(pm.ctx, cmd, args...)
	process.Dir = plug.Config.WorkingDir
	process.Stderr = os.Stderr
	process.Env = os.Environ()

	// Set up environment
	for k, v := range plug.Config.Environment {
		process.Env = append(process.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := process.Start(); err != nil {
		plug.LastError = fmt.Errorf("failed to restart plugin: %v", err)
		return
	}

	time.Sleep(time.Second)

	client, err := grpc.NewClient(plug.Config.Port)
	if err != nil {
		plug.LastError = fmt.Errorf("failed to reconnect to plugin: %v", err)
		return
	}

	grpcClient, ok := client.(*grpc.Client)
	if !ok {
		plug.LastError = fmt.Errorf("invalid client type after restart")
		return
	}

	plug.Client = client
	plug.GRPCClient = grpcClient
	plug.Cmd = process
}
