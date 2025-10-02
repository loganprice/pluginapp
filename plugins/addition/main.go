package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/example/grpc-plugin-app/pkg/plugin"
	"github.com/example/grpc-plugin-app/proto"
	"google.golang.org/grpc"
)

const (
	pluginVersion = "1.0.0"
)

// AdditionPlugin directly implements the proto.PluginServer interface
type AdditionPlugin struct {
	proto.UnimplementedPluginServer
}

// GetInfo implements the GetInfo RPC method
func (p *AdditionPlugin) GetInfo(ctx context.Context, req *proto.InfoRequest) (*proto.PluginInfo, error) {
	return &proto.PluginInfo{
		Name:        "addition",
		Version:     pluginVersion,
		Description: "A plugin that adds a series of numbers together",
		ParameterSpecs: map[string]*proto.ParamSpec{
			"num1": {
				Name:        "num1",
				Description: "First number to add",
				Required:    true,
				Type:        "float",
			},
			"num2": {
				Name:        "num2",
				Description: "Second number to add",
				Required:    true,
				Type:        "float",
			},
			"num3": {
				Name:        "num3",
				Description: "Third number to add (optional)",
				Required:    false,
				Type:        "float",
			},
			"num4": {
				Name:        "num4",
				Description: "Fourth number to add (optional)",
				Required:    false,
				Type:        "float",
			},
			"num5": {
				Name:        "num5",
				Description: "Fifth number to add (optional)",
				Required:    false,
				Type:        "float",
			},
		},
	}, nil
}

// validateParameters validates the input parameters
func (p *AdditionPlugin) validateParameters(params map[string]string) error {
	// Check for required parameters
	if _, ok := params["num1"]; !ok {
		return fmt.Errorf("missing required parameter: num1")
	}
	if _, ok := params["num2"]; !ok {
		return fmt.Errorf("missing required parameter: num2")
	}

	// Validate all number parameters
	for key, value := range params {
		if strings.HasPrefix(key, "num") {
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return fmt.Errorf("invalid number for %s: %v", key, err)
			}
		}
	}

	return nil
}

// Execute implements the Execute RPC method
func (p *AdditionPlugin) Execute(req *proto.ExecuteRequest, stream proto.Plugin_ExecuteServer) error {
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

	// Collect and sort all numbers from parameters
	var numbers []float64
	var keys []string

	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: "Collecting numbers...",
		},
	}); err != nil {
		return err
	}

	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Progress{
			Progress: &proto.Progress{
				Stage:           "Initialization",
				PercentComplete: 0,
				CurrentStep:     1,
				TotalSteps:      4,
			},
		},
	}); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Find all number parameters (num1, num2, num3, etc.)
	for key := range req.Params {
		if strings.HasPrefix(key, "num") {
			keys = append(keys, key)
		}
	}

	// Sort keys to maintain order (num1, num2, num3, etc.)
	sort.Strings(keys)

	if len(keys) == 0 {
		return stream.Send(&proto.ExecuteOutput{
			Content: &proto.ExecuteOutput_Error{
				Error: &proto.Error{
					Code:    "NO_NUMBERS",
					Message: "no numbers provided (use num1, num2, num3, etc.)",
				},
			},
		})
	}

	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Progress{
			Progress: &proto.Progress{
				Stage:           "Processing Input",
				PercentComplete: 25,
				CurrentStep:     2,
				TotalSteps:      4,
			},
		},
	}); err != nil {
		return err
	}

	// Convert all numbers
	for _, key := range keys {
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
			numStr := req.Params[key]
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return stream.Send(&proto.ExecuteOutput{
					Content: &proto.ExecuteOutput_Error{
						Error: &proto.Error{
							Code:    "INVALID_NUMBER",
							Message: fmt.Sprintf("invalid number for %s", key),
							Details: err.Error(),
						},
					},
				})
			}
			numbers = append(numbers, num)
			if err := stream.Send(&proto.ExecuteOutput{
				Content: &proto.ExecuteOutput_Output{
					Output: fmt.Sprintf("Added %s = %.2f", key, num),
				},
			}); err != nil {
				return err
			}
			time.Sleep(300 * time.Millisecond)
		}
	}

	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: "\nCalculating sum...",
		},
	}); err != nil {
		return err
	}

	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Progress{
			Progress: &proto.Progress{
				Stage:           "Calculating",
				PercentComplete: 50,
				CurrentStep:     3,
				TotalSteps:      4,
			},
		},
	}); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Calculate running sum
	var sum float64
	for i, num := range numbers {
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
			sum += num
			if i > 0 {
				if err := stream.Send(&proto.ExecuteOutput{
					Content: &proto.ExecuteOutput_Output{
						Output: fmt.Sprintf("Running total: %.2f + %.2f = %.2f", sum-num, num, sum),
					},
				}); err != nil {
					return err
				}

				if err := stream.Send(&proto.ExecuteOutput{
					Content: &proto.ExecuteOutput_Progress{
						Progress: &proto.Progress{
							Stage:           "Calculating",
							PercentComplete: 50 + float32(i)*25/float32(len(numbers)-1),
							CurrentStep:     3,
							TotalSteps:      4,
						},
					},
				}); err != nil {
					return err
				}
				time.Sleep(300 * time.Millisecond)
			}
		}
	}

	// Build the final output string
	var expression []string
	for _, num := range numbers {
		expression = append(expression, fmt.Sprintf("%.2f", num))
	}

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

	if err := stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: fmt.Sprintf("\nFinal result: %s = %.2f", strings.Join(expression, " + "), sum),
		},
	}); err != nil {
		return err
	}

	return nil
}

// ReportExecutionSummary implements the ReportExecutionSummary RPC method
func (p *AdditionPlugin) ReportExecutionSummary(ctx context.Context, req *proto.SummaryRequest) (*proto.SummaryResponse, error) {
	return &proto.SummaryResponse{
		PluginName: "addition",
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Duration:   float64(req.EndTime-req.StartTime) / float64(time.Millisecond),
		Success:    req.Success,
		Error:      req.Error,
		Metadata:   req.Metadata,
		Metrics:    req.Metrics,
	},
	nil
}

func main() {
	// Parse command line flags
	port := flag.Int("port", 0, "Port to listen on")
	flag.Parse()

	if *port == 0 {
		log.Fatal("Please specify a port using -port flag")
	}

	// Run the server
	server := grpc.NewServer()
	proto.RegisterPluginServer(server, &AdditionPlugin{})
	if err := plugin.RunGRPCServer(server, *port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
