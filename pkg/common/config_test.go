package common

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadPluginsConfig(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir) // Clean up the temporary directory

	// Test case 1: Valid config file
	validConfigContent := `
	{
		"plugins": [
			{
				"name": "test-plugin-1",
				"type": "binary",
				"path": "/path/to/plugin1",
				"env": {
					"ENV_VAR_1": "value1"
				}
			},
			{
				"name": "test-plugin-2",
				"type": "command",
				"command": "echo {port}",
				"path": "/path/to/plugin2"
			}
		]
	}`
	validConfigFile := filepath.Join(tmpDir, "valid_config.json")
	if err := os.WriteFile(validConfigFile, []byte(validConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write valid config file: %v", err)
	}

	// Test case 2: Invalid JSON config file
	invalidConfigContent := `{"plugins": [` // Malformed JSON
	invalidConfigFile := filepath.Join(tmpDir, "invalid_config.json")
	if err := os.WriteFile(invalidConfigFile, []byte(invalidConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	// Test case 3: Empty plugins array
	emptyPluginsContent := `{"plugins": []}`
	emptyPluginsFile := filepath.Join(tmpDir, "empty_plugins.json")
	if err := os.WriteFile(emptyPluginsFile, []byte(emptyPluginsContent), 0644); err != nil {
		t.Fatalf("Failed to write empty plugins config file: %v", err)
	}

	tests := []struct {
		name        string
		configPath  string
		wantErr     bool
		wantPlugins *PluginsConfig
	}{
		{
			name:       "Valid config file",
			configPath: validConfigFile,
			wantErr:    false,
			wantPlugins: &PluginsConfig{
				Plugins: []PluginConfig{
					{
						Name: "test-plugin-1",
						Type: "binary",
						Path: "/path/to/plugin1",
						Environment: map[string]string{
							"ENV_VAR_1": "value1",
						},
					},
					{
						Name:    "test-plugin-2",
						Type:    "command",
						Command: "echo {port}",
						Path:    "/path/to/plugin2",
					},
				},
			},
		},
		{
			name:        "File not found",
			configPath:  filepath.Join(tmpDir, "non_existent.json"),
			wantErr:     true,
			wantPlugins: nil,
		},
		{
			name:        "Invalid JSON",
			configPath:  invalidConfigFile,
			wantErr:     true,
			wantPlugins: nil,
		},
		{
			name:        "Empty plugins array",
			configPath:  emptyPluginsFile,
			wantErr:     false,
			wantPlugins: &PluginsConfig{Plugins: []PluginConfig{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadPluginsConfig(tt.configPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadPluginsConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantPlugins) {
				t.Errorf("LoadPluginsConfig() got = %v, want %v", got, tt.wantPlugins)
			}
		})
	}
}

func TestStartPlugin_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		config  PluginConfig
		port    int
		wantErr bool
	}{
		{
			name: "Unsupported plugin type",
			config: PluginConfig{
				Type: "unsupported",
			},
			port:    8080,
			wantErr: true,
		},
		{
			name: "Command type with empty command",
			config: PluginConfig{
				Type:    "command",
				Command: "",
			},
			port:    8080,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := StartPlugin(tt.config, tt.port)

			if (err != nil) != tt.wantErr {
				t.Errorf("StartPlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}