package shared

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/example/grpc-plugin-app/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Version information
const (
	APIVersion = "1.0.0"
)

// Feature flags for plugin capabilities
const (
	FeatureProgress     = "progress"
	FeatureParamSchema  = "param-schema"
	FeatureRichErrors   = "rich-errors"
	FeatureInteractive  = "interactive"
	FeatureCancellation = "cancellation"
)

// ExecutionSummary contains all information about a plugin's execution
type ExecutionSummary struct {
	PluginName string
	StartTime  int64
	EndTime    int64
	Duration   float64 // in milliseconds
	Success    bool
	Error      error
	Metadata   map[string]string
	Metrics    map[string]float64
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name            string
	Version         string
	Description     string
	ParameterSchema map[string]ParameterSpec
}

// ParameterSpec describes a plugin parameter
type ParameterSpec struct {
	Name          string
	Description   string
	Required      bool
	DefaultValue  string
	Type          string
	AllowedValues []string
}

// Progress represents execution progress information
type Progress struct {
	PercentComplete float32
	Stage           string
	CurrentStep     int32
	TotalSteps      int32
}

// OutputHandler handles different types of plugin output
type OutputHandler interface {
	OnOutput(msg string) error
	OnProgress(progress Progress) error
	OnError(code, message, details string) error
}

// PluginInterface is the interface that plugins must implement
type PluginInterface interface {
	GetInfo(ctx context.Context) (*PluginInfo, error)
	Execute(ctx context.Context, params map[string]string, output OutputHandler) error
	ReportExecutionSummary(startTime, endTime int64, success bool, err error, metadata map[string]string, metrics map[string]float64) (*ExecutionSummary, error)
	ValidateParameters(params map[string]string) error
	Close() error
}

// GRPCServer wraps the plugin implementation
type GRPCServer struct {
	proto.UnimplementedPluginServer
	Impl   PluginInterface
	server *grpc.Server
	done   chan struct{}
	wg     sync.WaitGroup
	name   string
}

// GetInfo implements the GetInfo RPC method
func (s *GRPCServer) GetInfo(ctx context.Context, req *proto.InfoRequest) (*proto.PluginInfo, error) {
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
func (s *GRPCServer) Execute(req *proto.ExecuteRequest, stream proto.Plugin_ExecuteServer) error {
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
	handler := &grpcOutputHandler{stream: stream}

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

// grpcOutputHandler implements OutputHandler for gRPC streaming
type grpcOutputHandler struct {
	stream proto.Plugin_ExecuteServer
}

func (h *grpcOutputHandler) OnOutput(msg string) error {
	return h.stream.Send(&proto.ExecuteOutput{
		Content: &proto.ExecuteOutput_Output{
			Output: msg,
		},
	})
}

func (h *grpcOutputHandler) OnProgress(p Progress) error {
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

func (h *grpcOutputHandler) OnError(code, message, details string) error {
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
func (s *GRPCServer) ReportExecutionSummary(ctx context.Context, req *proto.SummaryRequest) (*proto.SummaryResponse, error) {
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

// StartPluginServer starts the gRPC server for the plugin
func StartPluginServer(impl PluginInterface, port int) (chan struct{}, error) {
	// Listen on the specified port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %v", port, err)
	}

	server := grpc.NewServer()
	done := make(chan struct{})
	grpcServer := &GRPCServer{
		Impl:   impl,
		server: server,
		done:   done,
	}
	proto.RegisterPluginServer(server, grpcServer)

	go func() {
		if err := server.Serve(listener); err != nil {
			log.Printf("Server error: %v\n", err)
		}
	}()

	// Start a goroutine to handle graceful shutdown
	go func() {
		<-done
		grpcServer.wg.Wait() // Wait for any ongoing RPCs to complete
		server.GracefulStop()
	}()

	return done, nil
}

// NewPluginClient creates a new plugin client
func NewPluginClient(port int) (PluginInterface, error) {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to port %d: %v", port, err)
	}

	return &GRPCClient{
		client: proto.NewPluginClient(conn),
		conn:   conn,
	}, nil
}

// GRPCClient implements the PluginInterface for the client side
type GRPCClient struct {
	client proto.PluginClient
	conn   *grpc.ClientConn
	name   string
	info   *PluginInfo
}

// GetInfo retrieves plugin information
func (c *GRPCClient) GetInfo(ctx context.Context) (*PluginInfo, error) {
	if c.info != nil {
		return c.info, nil
	}

	resp, err := c.client.GetInfo(ctx, &proto.InfoRequest{})
	if err != nil {
		return nil, err
	}

	paramSchema := make(map[string]ParameterSpec)
	for name, spec := range resp.ParameterSpecs {
		paramSchema[name] = ParameterSpec{
			Name:          spec.Name,
			Description:   spec.Description,
			Required:      spec.Required,
			DefaultValue:  spec.DefaultValue,
			Type:          spec.Type,
			AllowedValues: spec.AllowedValues,
		}
	}

	c.info = &PluginInfo{
		Name:            resp.Name,
		Version:         resp.Version,
		Description:     resp.Description,
		ParameterSchema: paramSchema,
	}

	return c.info, nil
}

// ValidateParameters validates the parameters against the plugin's schema
func (c *GRPCClient) ValidateParameters(params map[string]string) error {
	info, err := c.GetInfo(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get plugin info: %v", err)
	}

	for name, spec := range info.ParameterSchema {
		value, exists := params[name]

		// Check required parameters
		if spec.Required && !exists {
			return fmt.Errorf("missing required parameter: %s", name)
		}

		if exists {
			// Check allowed values if specified
			if len(spec.AllowedValues) > 0 {
				valid := false
				for _, allowed := range spec.AllowedValues {
					if value == allowed {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("invalid value for %s: %s (allowed values: %v)", name, value, spec.AllowedValues)
				}
			}

			// Add type validation here if needed
		}
	}

	return nil
}

// Execute calls the Execute RPC method
func (c *GRPCClient) Execute(ctx context.Context, params map[string]string, handler OutputHandler) error {
	stream, err := c.client.Execute(ctx, &proto.ExecuteRequest{
		Params: params,
	})
	if err != nil {
		return fmt.Errorf("failed to start execution: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return fmt.Errorf("error receiving output: %v", err)
		}

		switch content := resp.Content.(type) {
		case *proto.ExecuteOutput_Output:
			if err := handler.OnOutput(content.Output); err != nil {
				return fmt.Errorf("error handling output: %v", err)
			}
		case *proto.ExecuteOutput_Error:
			return handler.OnError(content.Error.Code, content.Error.Message, content.Error.Details)
		case *proto.ExecuteOutput_Progress:
			if err := handler.OnProgress(Progress{
				PercentComplete: content.Progress.PercentComplete,
				Stage:           content.Progress.Stage,
				CurrentStep:     content.Progress.CurrentStep,
				TotalSteps:      content.Progress.TotalSteps,
			}); err != nil {
				return fmt.Errorf("error handling progress: %v", err)
			}
		}
	}
}

// ReportExecutionSummary sends execution summary to the main application
func (c *GRPCClient) ReportExecutionSummary(startTime, endTime int64, success bool, err error, metadata map[string]string, metrics map[string]float64) (*ExecutionSummary, error) {
	ctx := context.Background()
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	req := &proto.SummaryRequest{
		PluginName: c.name,
		StartTime:  startTime,
		EndTime:    endTime,
		Success:    success,
		Error:      errStr,
		Metadata:   metadata,
		Metrics:    metrics,
	}
	resp, err := c.client.ReportExecutionSummary(ctx, req)
	if err != nil {
		return nil, err
	}

	var execErr error
	if resp.Error != "" {
		execErr = fmt.Errorf(resp.Error)
	}

	return &ExecutionSummary{
		PluginName: resp.PluginName,
		StartTime:  resp.StartTime,
		EndTime:    resp.EndTime,
		Duration:   resp.Duration,
		Success:    resp.Success,
		Error:      execErr,
		Metadata:   resp.Metadata,
		Metrics:    resp.Metrics,
	}, nil
}

// Close closes the client connection
func (c *GRPCClient) Close() error {
	return c.conn.Close()
}
