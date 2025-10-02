
package process

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/example/grpc-plugin-app/pkg/plugin"
)

func TestStartPluginFromConfig(t *testing.T) {
	// Create a temporary directory for dummy executables
	tmpDir, err := os.MkdirTemp("", "start_plugin_from_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy executable that always fails to start
	failingBinaryPath := filepath.Join(tmpDir, "failing_binary")
	err = os.WriteFile(failingBinaryPath, []byte("#!/bin/bash\nexit 1"), 0755)
	if err != nil {
		t.Fatalf("Failed to create failing binary: %v", err)
	}

	// Create a dummy executable that succeeds
	succeedingBinaryPath := filepath.Join(tmpDir, "succeeding_binary")
	err = os.WriteFile(succeedingBinaryPath, []byte("#!/bin/bash\necho 'started'"), 0755)
	if err != nil {
		t.Fatalf("Failed to create succeeding binary: %v", err)
	}

	tests := []struct {
		name    string
		config  plugin.PluginConfig
		wantErr bool
	}{
		{
			name: "Failing plugin start (non-existent path)",
			config: plugin.PluginConfig{
				Path: filepath.Join(tmpDir, "non_existent_binary"),
				Port: 8080,
			},
			wantErr: true,
		},
		{
			name: "Successful plugin start (dummy)",
			config: plugin.PluginConfig{
				Path: succeedingBinaryPath,
				Port: 8081,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := StartPluginFromConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("StartPluginFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if cmd == nil {
					t.Errorf("StartPluginFromConfig() got nil cmd, want non-nil")
				} else {
					// Kill the process to clean up
					if cmd.Process != nil {
						cmd.Process.Kill()
					}
				}
			}
		})
	}
}
