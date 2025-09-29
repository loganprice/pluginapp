package common

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PluginConfig represents the configuration for a plugin
type PluginConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`    // "binary" or "command"
	Command     string            `json:"command"` // Command template with {port} placeholder
	Path        string            `json:"path"`    // Path to binary or command directory
	Environment map[string]string `json:"env"`     // Additional environment variables
	WorkingDir  string            `json:"workdir"` // Working directory for the command
}

// PluginsConfig represents the configuration for all plugins
type PluginsConfig struct {
	Plugins []PluginConfig `json:"plugins"`
}

// LoadPluginsConfig loads the plugin configuration from a JSON file
func LoadPluginsConfig(configPath string) (*PluginsConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config PluginsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

// StartPlugin starts a plugin using its configuration
func StartPlugin(config PluginConfig, port int) (*exec.Cmd, error) {
	switch config.Type {
	case "binary":
		// For Go binaries that use our standard flag
		cmdPath := filepath.Join(config.Path)
		cmd := exec.Command(cmdPath, "-port", fmt.Sprintf("%d", port))
		cmd.Dir = config.WorkingDir
		cmd.Env = os.Environ() // Start with current environment

		// Add additional environment variables
		for k, v := range config.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd, cmd.Start()
	case "command":
		// Replace {port} in command template
		cmdStr := strings.ReplaceAll(config.Command, "{port}", fmt.Sprintf("%d", port))

		// Split command into parts
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			return nil, fmt.Errorf("empty command")
		}

		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = config.WorkingDir
		cmd.Env = os.Environ()

		// Add additional environment variables
		for k, v := range config.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd, cmd.Start()
	}

	return nil, fmt.Errorf("unsupported plugin type: %s", config.Type)
}
