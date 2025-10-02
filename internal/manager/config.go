package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/example/grpc-plugin-app/pkg/plugin"
)

// AppConfig represents the main application configuration
type AppConfig struct {
	Plugins map[string]plugin.PluginConfig `json:"plugins"`
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(configPath string) (*AppConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Get workspace root (where config.json is)
	workspaceRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace root: %v", err)
	}

	// Resolve relative paths and set defaults
	for name, pluginConfig := range config.Plugins {
		// Resolve relative paths
		if !filepath.IsAbs(pluginConfig.Path) {
			pluginConfig.Path = filepath.Join(workspaceRoot, pluginConfig.Path)
		}
		if pluginConfig.WorkingDir != "" && !filepath.IsAbs(pluginConfig.WorkingDir) {
			pluginConfig.WorkingDir = filepath.Join(workspaceRoot, pluginConfig.WorkingDir)
		}

		// Set defaults
		if pluginConfig.Type == "" {
			pluginConfig.Type = plugin.PluginTypeBinary // Default to Go binary for backward compatibility
		}
		if pluginConfig.Environment == nil {
			pluginConfig.Environment = make(map[string]string)
		}
		if pluginConfig.WorkingDir == "" {
			pluginConfig.WorkingDir = filepath.Dir(pluginConfig.Path)
		}
		if pluginConfig.Defaults == nil {
			pluginConfig.Defaults = make(map[string]string)
		}

		// Validate the configuration
		if err := pluginConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration for plugin %q: %v", name, err)
		}

		config.Plugins[name] = pluginConfig
	}

	return &config, nil
}

// GetPluginConfig retrieves the configuration for a specific plugin
func (c *AppConfig) GetPluginConfig(name string) (plugin.PluginConfig, error) {
	if plugin, ok := c.Plugins[name]; ok {
		return plugin, nil
	}
	return plugin.PluginConfig{}, fmt.Errorf("plugin %q not found in configuration", name)
}

// ListPlugins returns a list of all configured plugins with their descriptions
func (c *AppConfig) ListPlugins() []string {
	var result []string
	for name, plugin := range c.Plugins {
		result = append(result, fmt.Sprintf("%s: %s", name, plugin.Description))
	}
	return result
}

// SaveConfig saves the configuration to the specified file
func SaveConfig(config *AppConfig, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}
