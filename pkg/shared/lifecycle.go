package shared

import (
	"context"
	"fmt"
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

	// Create a copy of the plugin config to avoid race conditions
	config := pluginConfig

	// Get the appropriate start command based on plugin type
	cmd, args, err := config.GetStartCommand(config.Port)
	if err != nil {
		return fmt.Errorf("failed to get start command: %v", err)
	}

	// Start the plugin process
	process := exec.CommandContext(pm.ctx, cmd, args...)
	process.Dir = config.WorkingDir
	process.Stderr = os.Stderr
	process.Stdout = os.Stdout

	// Set up environment
	process.Env = os.Environ()
	for k, v := range config.Environment {
		process.Env = append(process.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := process.Start(); err != nil {
		return fmt.Errorf("failed to start plugin %s: %v", name, err)
	}

	// Wait for the plugin to start and be ready
	var client PluginInterface
	var clientErr error
	for retries := 0; retries < 5; retries++ {
		time.Sleep(time.Second)
		client, clientErr = NewPluginClient(config.Port)
		if clientErr == nil {
			break
		}
	}

	if clientErr != nil {
		process.Process.Kill()
		return fmt.Errorf("failed to connect to plugin %s after multiple attempts: %v", name, clientErr)
	}

	grpcClient, ok := client.(*GRPCClient)
	if !ok {
		process.Process.Kill()
		return fmt.Errorf("invalid client type for plugin %s", name)
	}

	// Set the plugin name in the client for telemetry
	grpcClient.name = name

	managed := &ManagedPlugin{
		Name:       name,
		Config:     config,
		Client:     client,
		GRPCClient: grpcClient,
		Cmd:        process,
	}

	// Enable health checking with automatic restart
	grpcClient.EnableHealthCheck(pm.ctx, HealthCheck{
		Interval:   time.Second * 30,
		MaxRetries: 3,
		RetryDelay: time.Second * 5,
		OnUnhealthy: func(err error) {
			pm.mu.Lock()
			defer pm.mu.Unlock()

			managed.LastError = err
			if managed.RestartCnt < 3 {
				managed.RestartCnt++
				pm.restartPlugin(managed)
			}
		},
	})

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
		return fmt.Errorf("failed to close plugin client: %v", err)
	}

	if err := plugin.Cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill plugin process: %v", err)
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
		plugin.Cmd.Process.Kill()
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
