package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PluginType represents the type of plugin
type PluginType string

const (
	// PluginTypeBinary represents a Go binary plugin
	PluginTypeBinary PluginType = "binary"
	// PluginTypeCommand represents a plugin started with a custom command
	PluginTypeCommand PluginType = "command"
	// PluginTypeRemote represents a plugin running on a remote server
	PluginTypeRemote PluginType = "remote"
)

// PluginConfig represents the configuration for a plugin
type PluginConfig struct {
	Path        string            `json:"path,omitempty"`
	Port        int               `json:"port,omitempty"`
	Type        PluginType        `json:"type"`
	Command     string            `json:"command,omitempty"`
	Address     string            `json:"address,omitempty"`
	Description string            `json:"description"`
	Defaults    map[string]string `json:"defaults,omitempty"`
	WorkingDir  string            `json:"workdir,omitempty"`
	Environment map[string]string `json:"env,omitempty"`
}

// Validate checks if the plugin configuration is valid
func (p *PluginConfig) Validate() error {
	switch p.Type {
	case PluginTypeBinary, PluginTypeCommand:
		if p.Path == "" {
			return fmt.Errorf("path is required for %s type plugins", p.Type)
		}
		if p.Port <= 0 {
			return fmt.Errorf("invalid port for local plugin: %d", p.Port)
		}
	case PluginTypeRemote:
		if p.Address == "" {
			return fmt.Errorf("address is required for remote-type plugins")
		}
	default:
		return fmt.Errorf("unsupported plugin type: %s", p.Type)
	}

	if p.Type == PluginTypeCommand {
		if p.Command == "" {
			return fmt.Errorf("command is required for command-type plugins")
		}
		if !strings.Contains(p.Command, "{port}") {
			return fmt.Errorf("command must contain {port} placeholder")
		}
	}

	return nil
}

// AppConfig represents the main application configuration
type AppConfig struct {
	Plugins map[string]PluginConfig `json:"plugins"`
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
	for name, plugin := range config.Plugins {
		// Resolve relative paths
		if !filepath.IsAbs(plugin.Path) {
			plugin.Path = filepath.Join(workspaceRoot, plugin.Path)
		}
		if plugin.WorkingDir != "" && !filepath.IsAbs(plugin.WorkingDir) {
			plugin.WorkingDir = filepath.Join(workspaceRoot, plugin.WorkingDir)
		}

		// Set defaults
		if plugin.Type == "" {
			plugin.Type = PluginTypeBinary // Default to Go binary for backward compatibility
		}
		if plugin.Environment == nil {
			plugin.Environment = make(map[string]string)
		}
		if plugin.WorkingDir == "" {
			plugin.WorkingDir = filepath.Dir(plugin.Path)
		}
		if plugin.Defaults == nil {
			plugin.Defaults = make(map[string]string)
		}

		// Validate the configuration
		if err := plugin.Validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration for plugin %q: %v", name, err)
		}

		config.Plugins[name] = plugin
	}

	return &config, nil
}

// GetPluginConfig retrieves the configuration for a specific plugin
func (c *AppConfig) GetPluginConfig(name string) (PluginConfig, error) {
	if plugin, ok := c.Plugins[name]; ok {
		return plugin, nil
	}
	return PluginConfig{}, fmt.Errorf("plugin %q not found in configuration", name)
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

// GetStartCommand returns the appropriate command to start the plugin
func (p *PluginConfig) GetStartCommand(port int) (string, []string, error) {
	switch p.Type {
	case PluginTypeBinary:
		return p.Path, []string{"-port", fmt.Sprintf("%d", port)}, nil
	case PluginTypeCommand:
		if p.Command == "" {
			return "", nil, fmt.Errorf("command template not specified for command-type plugin")
		}

		// Replace both port and path placeholders
		cmd := strings.ReplaceAll(p.Command, "{port}", fmt.Sprintf("%d", port))
		cmd = strings.ReplaceAll(cmd, "{path}", p.Path)

		// Split into command and arguments
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return "", nil, fmt.Errorf("empty command after template substitution")
		}

		return parts[0], parts[1:], nil
	default:
		return "", nil, fmt.Errorf("unsupported plugin type: %s", p.Type)
	}
}
