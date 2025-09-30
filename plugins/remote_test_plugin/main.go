package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/example/grpc-plugin-app/pkg/shared"
)

// RemoteTestPlugin implements the PluginInterface
type RemoteTestPlugin struct{}

// GetInfo returns information about the plugin
func (p *RemoteTestPlugin) GetInfo(ctx context.Context) (*shared.PluginInfo, error) {
	return &shared.PluginInfo{
		Name:        "remote-test-plugin",
		Version:     "1.0.0",
		Description: "A plugin to test remote functionality.",
		ParameterSchema: map[string]shared.ParameterSpec{
			"message": {
				Name:        "message",
				Description: "A message to be echoed back.",
				Required:    true,
			},
		},
	}, nil
}

// Execute runs the plugin's logic
func (p *RemoteTestPlugin) Execute(ctx context.Context, params map[string]string, handler shared.OutputHandler) error {
	message, ok := params["message"]
	if !ok {
		return handler.OnError("MISSING_PARAM", "Missing required parameter: message", "")
	}

	info, _ := p.GetInfo(ctx)

	handler.OnOutput(fmt.Sprintf("Hello from the %s!", info.Name))
	handler.OnOutput(fmt.Sprintf("I received the message: '%s'", message))

	return nil
}

// ReportExecutionSummary is a no-op for this simple plugin
func (p *RemoteTestPlugin) ReportExecutionSummary(startTime, endTime int64, success bool, err error, metadata map[string]string, metrics map[string]float64) (*shared.ExecutionSummary, error) {
	return &shared.ExecutionSummary{
		PluginName: "remote-test-plugin",
		Success:    success,
		Error:      err,
	}, nil
}

// ValidateParameters is a no-op for this simple plugin
func (p *RemoteTestPlugin) ValidateParameters(params map[string]string) error {
	return nil
}

// Close is a no-op
func (p *RemoteTestPlugin) Close() error {
	return nil
}

func main() {
	port := flag.Int("port", 50055, "Port for the gRPC server to listen on")
	flag.Parse()

	log.Printf("Starting remote test plugin on port %d...", *port)

	impl := &RemoteTestPlugin{}
	done, err := shared.StartPluginServer(impl, *port)
	if err != nil {
		log.Fatalf("Failed to start plugin server: %v", err)
	}

	// Wait for a signal to stop the server
	<-done
	log.Println("Plugin server stopped.")
}
