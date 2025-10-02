package plugin

import (
	"fmt"
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

// GetStartCommand returns the appropriate command to start the plugin
func (p *PluginConfig) GetStartCommand(port int, args map[string]string) (string, []string, error) {
	// Convert the map to a slice of --key=value strings
	var argSlice []string
	for k, v := range args {
		argSlice = append(argSlice, fmt.Sprintf("--%s=%s", k, v))
	}
	argString := strings.Join(argSlice, " ")

	switch p.Type {
	case PluginTypeBinary:
		finalArgs := append([]string{"-port", fmt.Sprintf("%d", port)}, argSlice...)
		return p.Path, finalArgs, nil

	case PluginTypeCommand:
		if p.Command == "" {
			return "", nil, fmt.Errorf("command template not specified for command-type plugin")
		}

		// Replace placeholders
		cmd := strings.ReplaceAll(p.Command, "{port}", fmt.Sprintf("%d", port))
		cmd = strings.ReplaceAll(cmd, "{path}", p.Path)
		cmd = strings.ReplaceAll(cmd, "{args}", argString)

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
