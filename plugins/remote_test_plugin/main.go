package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/example/grpc-plugin-app/proto"
	"google.golang.org/grpc"
)

// RemoteTestPlugin implements the PluginInterface
type RemoteTestPlugin struct{
	proto.UnimplementedPluginServer
}

// GetInfo returns information about the plugin
func (p *RemoteTestPlugin) GetInfo(ctx context.Context, req *proto.InfoRequest) (*proto.PluginInfo, error) {
	return &proto.PluginInfo{
		Name:        "remote-test-plugin",
		Version:     "1.0.0",
		Description: "A plugin to test remote functionality.",
		ParameterSpecs: map[string]*proto.ParamSpec{
			"message": {
				Name:        "message",
				Description: "A message to be echoed back.",
				Required:    true,
			},
		},
	}, nil
}

// Execute runs the plugin's logic
func (p *RemoteTestPlugin) Execute(req *proto.ExecuteRequest, stream proto.Plugin_ExecuteServer) error {
	message, ok := req.Params["message"]
	if !ok {
		return stream.Send(&proto.ExecuteOutput{
			Content: &proto.ExecuteOutput_Error{
				Error: &proto.Error{
					Code:    "MISSING_PARAM",
					Message: "Missing required parameter: message",
				},
			},
		})
	}

	info, _ := p.GetInfo(stream.Context(), nil)

	stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: fmt.Sprintf("Hello from the %s!", info.Name),
		},
	})
	stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: fmt.Sprintf("I received the message: '%s'", message),
		},
	})

	return nil
}

// ReportExecutionSummary is a no-op for this simple plugin
func (p *RemoteTestPlugin) ReportExecutionSummary(ctx context.Context, req *proto.SummaryRequest) (*proto.SummaryResponse, error) {
	return &proto.SummaryResponse{}, nil
}

func main() {
	port := flag.Int("port", 50055, "Port for the gRPC server to listen on")
	flag.Parse()

	log.Printf("Starting remote test plugin on port %d...", *port)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", *port, err)
	}

	server := grpc.NewServer()
	impl := &RemoteTestPlugin{}
	proto.RegisterPluginServer(server, impl)

	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
