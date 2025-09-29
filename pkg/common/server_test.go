package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/example/grpc-plugin-app/pkg/shared"
	"github.com/example/grpc-plugin-app/proto"
)

// MockPluginServer is a mock implementation of proto.PluginServer for testing
type MockPluginServer struct {
	proto.UnimplementedPluginServer
}

func TestRunGRPCServer(t *testing.T) {
	tests := []struct {
		name    string
		plugin  proto.PluginServer
		port    int
		wantErr bool
	}{
		{
			name:    "Invalid port",
			plugin:  &MockPluginServer{},
			port:    0,
			wantErr: true,
		},
		// TODO: Add test cases for successful server start and listen errors.
		// These are complex to test without mocking net.Listen and grpc.Server.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunGRPCServer(tt.plugin, tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunGRPCServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
		config  shared.PluginConfig
		wantErr bool
	}{
		{
			name: "Failing plugin start (non-existent path)",
			config: shared.PluginConfig{
				Path: filepath.Join(tmpDir, "non_existent_binary"),
				Port: 8080,
			},
			wantErr: true,
		},
		{
			name: "Successful plugin start (dummy)",
			config: shared.PluginConfig{
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
