package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/example/grpc-plugin-app/pkg/common"
	"github.com/example/grpc-plugin-app/proto"
)

const (
	pluginVersion = "1.0.0"
)

// HelloPlugin directly implements the proto.PluginServer interface
type HelloPlugin struct {
	proto.UnimplementedPluginServer
}

// GetInfo implements the GetInfo RPC method
func (p *HelloPlugin) GetInfo(ctx context.Context, req *proto.InfoRequest) (*proto.PluginInfo, error) {
	return &proto.PluginInfo{
		Name:        "hello",
		Version:     pluginVersion,
		Description: "A friendly plugin that greets you",
		ParameterSpecs: map[string]*proto.ParamSpec{
			"message": {
				Name:         "message",
				Description:  "The name or message to greet",
				Required:     false,
				DefaultValue: "World",
				Type:         "string",
			},
			"language": {
				Name:          "language",
				Description:   "The language to use for greeting",
				Required:      false,
				DefaultValue:  "en",
				Type:          "string",
				AllowedValues: []string{"en", "es", "fr", "de"},
			},
		},
	}, nil
}

// validateParameters validates the input parameters
func (p *HelloPlugin) validateParameters(params map[string]string) error {
	// Check language if specified
	if lang, ok := params["language"]; ok {
		validLangs := map[string]bool{
			"en": true,
			"es": true,
			"fr": true,
			"de": true,
		}
		if !validLangs[lang] {
			return fmt.Errorf("unsupported language: %s (supported: en, es, fr, de)", lang)
		}
	}
	return nil
}

// Execute implements the Execute RPC method
func (p *HelloPlugin) Execute(req *proto.ExecuteRequest, stream proto.Plugin_ExecuteServer) error {
	// Validate parameters
	if err := p.validateParameters(req.Params); err != nil {
		return stream.Send(&proto.ExecuteOutput{
			Content: &proto.ExecuteOutput_Error{
				Error: &proto.Error{
					Code:    "INVALID_PARAMETERS",
					Message: err.Error(),
				},
			},
		})
	}

	// Get message parameter with default
	message := req.Params["message"]
	if message == "" {
		message = "World"
	}

	// Get language parameter with default
	language := req.Params["language"]
	if language == "" {
		language = "en"
	}

	// Report initial progress
	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Progress{
			Progress: &proto.Progress{
				Stage:           "Starting",
				PercentComplete: 0,
				CurrentStep:     1,
				TotalSteps:      4,
			},
		},
	}); err != nil {
		return err
	}

	// Send initial message
	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: fmt.Sprintf("Starting to greet %s in %s...", message, language),
		},
	}); err != nil {
		return err
	}
	time.Sleep(time.Second)

	// Report progress before dots
	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Progress{
			Progress: &proto.Progress{
				Stage:           "Processing",
				PercentComplete: 25,
				CurrentStep:     2,
				TotalSteps:      4,
			},
		},
	}); err != nil {
		return err
	}

	// Send some dots to show progress
	dots := 0
	for i := 0; i < 3; i++ {
		select {
		case <-stream.Context().Done():
			return stream.Send(&proto.ExecuteOutput{
				Content: &proto.ExecuteOutput_Error{
					Error: &proto.Error{
						Code:    "CANCELLED",
						Message: "Operation cancelled by user",
						Details: stream.Context().Err().Error(),
					},
				},
			})
		default:
			// Send a dot
			if err := stream.Send(&proto.ExecuteOutput{
				Content: &proto.ExecuteOutput_Output{
					Output: "...",
				},
			}); err != nil {
				return err
			}
			dots++

			// Update progress during dots
			if err := stream.Send(&proto.ExecuteOutput{
				Content: &proto.ExecuteOutput_Progress{
					Progress: &proto.Progress{
						Stage:           "Processing",
						PercentComplete: 25 + float32(i+1)*25,
						CurrentStep:     int32(2 + i),
						TotalSteps:      4,
					},
				},
			}); err != nil {
				return err
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	// Prepare final greeting based on language
	var greeting string
	switch language {
	case "es":
		greeting = fmt.Sprintf("¡Hola, %s!", message)
	case "fr":
		greeting = fmt.Sprintf("Bonjour, %s!", message)
	case "de":
		greeting = fmt.Sprintf("Hallo, %s!", message)
	default:
		greeting = fmt.Sprintf("Hello, %s!", message)
	}

	// Send final progress
	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Progress{
			Progress: &proto.Progress{
				Stage:           "Finalizing",
				PercentComplete: 100,
				CurrentStep:     4,
				TotalSteps:      4,
			},
		},
	}); err != nil {
		return err
	}

	// Send the final greeting
	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: greeting,
		},
	}); err != nil {
		return err
	}

	return nil
}

// ReportExecutionSummary implements the ReportExecutionSummary RPC method
func (p *HelloPlugin) ReportExecutionSummary(ctx context.Context, req *proto.SummaryRequest) (*proto.SummaryResponse, error) {
	return &proto.SummaryResponse{
		PluginName: "hello",
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Duration:   float64(req.EndTime-req.StartTime) / float64(time.Millisecond),
		Success:    req.Success,
		Error:      req.Error,
		Metadata:   req.Metadata,
		Metrics:    req.Metrics,
	}, nil
}

func main() {
	// Parse command line flags
	port := flag.Int("port", 0, "Port to listen on")
	flag.Parse()

	if *port == 0 {
		log.Fatal("Please specify a port using -port flag")
	}

	// Run the server
	if err := common.RunGRPCServer(&HelloPlugin{}, *port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
