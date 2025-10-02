
package grpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/example/grpc-plugin-app/pkg/plugin"
	"github.com/example/grpc-plugin-app/proto"
	"google.golang.org/grpc"
)

// Server wraps the plugin implementation
type Server struct {
	proto.UnimplementedPluginServer
	Impl   plugin.Plugin
	server *grpc.Server
	done   chan struct{}
	wg     sync.WaitGroup
	name   string
}

// GetInfo implements the GetInfo RPC method
func (s *Server) GetInfo(ctx context.Context, req *proto.InfoRequest) (*proto.PluginInfo, error) {
	info, err := s.Impl.GetInfo(ctx)
	if err != nil {
		return nil, err
	}

	paramSpecs := make(map[string]*proto.ParamSpec)
	for name, spec := range info.ParameterSchema {
		paramSpecs[name] = &proto.ParamSpec{
			Name:          spec.Name,
			Description:   spec.Description,
			Required:      spec.Required,
			DefaultValue:  spec.DefaultValue,
			Type:          spec.Type,
			AllowedValues: spec.AllowedValues,
		}
	}

	return &proto.PluginInfo{
		Name:           info.Name,
		Version:        info.Version,
		Description:    info.Description,
		ParameterSpecs: paramSpecs,
	}, nil
}

// Execute implements the Execute RPC method
func (s *Server) Execute(req *proto.ExecuteRequest, stream proto.Plugin_ExecuteServer) error {
	ctx := stream.Context()
	s.wg.Add(1)
	defer s.wg.Done()

	// Validate parameters first
	if err := s.Impl.ValidateParameters(req.Params); err != nil {
		return stream.Send(&proto.ExecuteOutput{
			Content: &proto.ExecuteOutput_Error{
				Error: &proto.Error{
					Code:    "INVALID_PARAMETERS",
					Message: err.Error(),
				},
			},
		})
	}

	// Create an output handler that sends messages through the stream
	handler := &outputHandler{stream: stream}

	// Execute the plugin
	if err := s.Impl.Execute(ctx, req.Params, handler); err != nil {
		// Only send error if it hasn't been sent through the handler
		if _, ok := err.(*handledError); !ok {
			return stream.Send(&proto.ExecuteOutput{
				Content: &proto.ExecuteOutput_Error{
					Error: &proto.Error{
						Code:    "EXECUTION_ERROR",
						Message: err.Error(),
					},
				},
			})
		}
		return err
	}

	return nil
}

// handledError indicates an error that's already been sent through the output handler
type handledError struct {
	err error
}

func (e *handledError) Error() string {
	return e.err.Error()
}

// outputHandler implements OutputHandler for gRPC streaming
type outputHandler struct {
	stream proto.Plugin_ExecuteServer
}

func (h *outputHandler) OnOutput(msg string) error {
	return h.stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: msg,
		},
	})
}

func (h *outputHandler) OnProgress(p plugin.Progress) error {
	return h.stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Progress{
			Progress: &proto.Progress{
				PercentComplete: p.PercentComplete,
				Stage:           p.Stage,
				CurrentStep:     p.CurrentStep,
				TotalSteps:      p.TotalSteps,
			},
		},
	})
}

func (h *outputHandler) OnError(code, message, details string) error {
	err := h.stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Error{
			Error: &proto.Error{
				Code:    code,
				Message: message,
				Details: details,
			},
		},
	})
	if err != nil {
		return err
	}
	return &handledError{fmt.Errorf("%s: %s", code, message)}
}

// ReportExecutionSummary implements the ReportExecutionSummary RPC method
func (s *Server) ReportExecutionSummary(ctx context.Context, req *proto.SummaryRequest) (*proto.SummaryResponse, error) {
	summary, err := s.Impl.ReportExecutionSummary(
		req.StartTime,
		req.EndTime,
		req.Success,
		fmt.Errorf(req.Error),
		req.Metadata,
		req.Metrics,
	)
	if err != nil {
		return nil, err
	}

	errStr := ""
	if summary.Error != nil {
		errStr = summary.Error.Error()
	}

	return &proto.SummaryResponse{
		PluginName: summary.PluginName,
		StartTime:  summary.StartTime,
		EndTime:    summary.EndTime,
		Duration:   summary.Duration,
		Success:    summary.Success,
		Error:      errStr,
		Metadata:   summary.Metadata,
		Metrics:    summary.Metrics,
	}, nil
}
