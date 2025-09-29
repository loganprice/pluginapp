package common

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/example/grpc-plugin-app/pkg/shared"
	"github.com/example/grpc-plugin-app/proto"
	"google.golang.org/grpc"
)

// RunGRPCServer initializes and runs a gRPC server for a plugin
func RunGRPCServer(plugin proto.PluginServer, port int) error {
	if port <= 0 {
		return fmt.Errorf("invalid port: %d", port)
	}

	// Create and configure gRPC server
	server := grpc.NewServer()
	proto.RegisterPluginServer(server, plugin)

	// Add health checking
	shared.StartHealthServer(server)

	// Listen on specified port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	// Start serving
	log.Printf("Starting plugin server on port %d\n", port)
	return server.Serve(listener)
}

// StartPluginFromConfig starts a plugin using the shared configuration
func StartPluginFromConfig(config shared.PluginConfig) (*exec.Cmd, error) {
	// Start the plugin process
	cmd := exec.Command(config.Path, "-port", fmt.Sprintf("%d", config.Port))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start plugin: %v", err)
	}

	return cmd, nil
}
