package shared

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
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
	Config     PluginConfig
	Client     PluginInterface
	GRPCClient *GRPCClient
	Cmd        *exec.Cmd
	RestartCnt int
	LastError  error
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
func (pm *PluginManager) StartPlugin(name string, pluginConfig PluginConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s is already running", name)
	}

	config := pluginConfig

	var client PluginInterface
	var clientErr error
	var process *exec.Cmd

	if config.Type == PluginTypeRemote {
		// For remote plugins, just connect, don't start a process
		client, clientErr = NewPluginClientWithAddress(config.Address)
	} else {
		// For local plugins, start the process and then connect
		cmd, args, err := config.GetStartCommand(config.Port)
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
			client, clientErr = NewPluginClient(config.Port)
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

	grpcClient, ok := client.(*GRPCClient)
	if !ok {
		if process != nil {
			process.Process.Kill()
		}
		return fmt.Errorf("invalid client type for plugin %s", name)
	}

	grpcClient.name = name

	managed := &ManagedPlugin{
		Name:       name,
		Config:     config,
		Client:     client,
		GRPCClient: grpcClient,
		Cmd:        process, // Cmd will be nil for remote plugins
	}

	// For local plugins, enable health checking with automatic restart
	if managed.Cmd != nil {
		// Note: HealthCheck is not fully implemented in the provided code
		// grpcClient.EnableHealthCheck(pm.ctx, HealthCheck{ ... })
	}

	pm.plugins[name] = managed
	return nil
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
func (pm *PluginManager) GetPlugin(name string) (PluginInterface, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s is not running", name)
	}

	return plugin.Client, nil
}

// restartPlugin attempts to restart a failed plugin
func (pm *PluginManager) restartPlugin(plugin *ManagedPlugin) {
	if plugin.Cmd == nil {
		plugin.LastError = fmt.Errorf("cannot restart a non-local plugin")
		return
	}

	plugin.Client.Close()
	plugin.Cmd.Process.Kill()

	// Get the appropriate start command based on plugin type
	cmd, args, err := plugin.Config.GetStartCommand(plugin.Config.Port)
	if err != nil {
		plugin.LastError = fmt.Errorf("failed to get restart command: %v", err)
		return
	}

	process := exec.CommandContext(pm.ctx, cmd, args...)
	process.Dir = plugin.Config.WorkingDir
	process.Stderr = os.Stderr
	process.Env = os.Environ()

	// Set up environment
	for k, v := range plugin.Config.Environment {
		process.Env = append(process.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := process.Start(); err != nil {
		plugin.LastError = fmt.Errorf("failed to restart plugin: %v", err)
		return
	}

	time.Sleep(time.Second)

	client, err := NewPluginClient(plugin.Config.Port)
	if err != nil {
		plugin.LastError = fmt.Errorf("failed to reconnect to plugin: %v", err)
		return
	}

	grpcClient, ok := client.(*GRPCClient)
	if !ok {
		plugin.LastError = fmt.Errorf("invalid client type after restart")
		return
	}

	plugin.Client = client
	plugin.GRPCClient = grpcClient
	plugin.Cmd = process
}