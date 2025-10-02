package plugin

import (
	"strings"
	"testing"
)

func TestPluginConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  PluginConfig
		wantErr bool
		errorMsg  string // Expected error message substring
	}{
		{
			name: "Valid Binary type",
			config: PluginConfig{
				Path: "/path/to/binary",
				Port: 8080,
				Type: PluginTypeBinary,
			},
			wantErr: false,
		},
		{
			name: "Valid Command type",
			config: PluginConfig{
				Path:    "/path/to/command",
				Port:    8081,
				Type:    PluginTypeCommand,
				Command: "mycommand {port}",
			},
			wantErr: false,
		},
		{
			name: "Missing Path",
			config: PluginConfig{
				Port: 8080,
				Type: PluginTypeBinary,
			},
			wantErr: true,
			errorMsg:  "path is required",
		},
		{
			name: "Invalid Port (zero)",
			config: PluginConfig{
				Path: "/path/to/binary",
				Port: 0,
				Type: PluginTypeBinary,
			},
			wantErr: true,
			errorMsg:  "invalid port",
		},
		{
			name: "Invalid Port (negative)",
			config: PluginConfig{
				Path: "/path/to/binary",
				Port: -1,
				Type: PluginTypeBinary,
			},
			wantErr: true,
			errorMsg:  "invalid port",
		},
		{
			name: "Command type, missing Command",
			config: PluginConfig{
				Path: "/path/to/command",
				Port: 8081,
				Type: PluginTypeCommand,
				Command: "",
			},
			wantErr: true,
			errorMsg:  "command is required for command-type plugins",
		},
		{
			name: "Command type, missing {port} in Command",
			config: PluginConfig{
				Path:    "/path/to/command",
				Port:    8081,
				Type:    PluginTypeCommand,
				Command: "mycommand without port",
			},
			wantErr: true,
			errorMsg:  "command must contain {port} placeholder",
		},
		{
			name: "Unsupported Plugin Type",
			config: PluginConfig{
				Path: "/path/to/plugin",
				Port: 8080,
				Type: "unknown_type",
			},
			wantErr: true,
			errorMsg:  "unsupported plugin type: unknown_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("PluginConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("PluginConfig.Validate() error message = %q, want substring %q", err.Error(), tt.errorMsg)
			}
		})
	}
}
