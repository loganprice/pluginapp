package plugin

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// RunGRPCServer initializes and runs a gRPC server for a plugin
func RunGRPCServer(server *grpc.Server, port int) error {
	if port <= 0 {
		return fmt.Errorf("invalid port: %d", port)
	}

	// Add health checking
	StartHealthServer(server)

	// Listen on specified port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	// Start serving
	log.Printf("Starting plugin server on port %d\n", port)
	return server.Serve(listener)
}

// StartHealthServer starts the gRPC health checking server
func StartHealthServer(server *grpc.Server) *health.Server {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	return healthServer
}
