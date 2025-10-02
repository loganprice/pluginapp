
package grpc

import (
	"context"
	"fmt"

	"github.com/example/grpc-plugin-app/pkg/plugin"
	"github.com/example/grpc-plugin-app/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client implements the plugin.Plugin for the client side
type Client struct {
	Client proto.PluginClient
	Conn   *grpc.ClientConn
	Name   string
	Info   *plugin.PluginInfo
}

func NewClient(port int) (plugin.Plugin, error) {
	address := fmt.Sprintf("localhost:%d", port)
	return NewClientWithAddress(address)
}

// NewClientWithAddress creates a new plugin client that connects to a specific address
func NewClientWithAddress(address string) (plugin.Plugin, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to address %s: %v", address, err)
	}

	return &Client{
		Client: proto.NewPluginClient(conn),
		Conn:   conn,
	}, nil
}

// GetInfo retrieves plugin information
func (c *Client) GetInfo(ctx context.Context) (*plugin.PluginInfo, error) {
	if c.Info != nil {
		return c.Info, nil
	}

	resp, err := c.Client.GetInfo(ctx, &proto.InfoRequest{})
	if err != nil {
		return nil, err
	}

	paramSchema := make(map[string]plugin.ParameterSpec)
	for name, spec := range resp.ParameterSpecs {
		paramSchema[name] = plugin.ParameterSpec{
			Name:          spec.Name,
			Description:   spec.Description,
			Required:      spec.Required,
			DefaultValue:  spec.DefaultValue,
			Type:          spec.Type,
			AllowedValues: spec.AllowedValues,
		}
	}

	c.Info = &plugin.PluginInfo{
		Name:            resp.Name,
		Version:         resp.Version,
		Description:     resp.Description,
		ParameterSchema: paramSchema,
	}

	return c.Info, nil
}

// ValidateParameters validates the parameters against the plugin's schema
func (c *Client) ValidateParameters(params map[string]string) error {
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
func (c *Client) Execute(ctx context.Context, params map[string]string, handler plugin.OutputHandler) error {
	stream, err := c.Client.Execute(ctx, &proto.ExecuteRequest{
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
			if err := handler.OnProgress(plugin.Progress{
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
func (c *Client) ReportExecutionSummary(startTime, endTime int64, success bool, err error, metadata map[string]string, metrics map[string]float64) (*plugin.ExecutionSummary, error) {
	ctx := context.Background()
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	req := &proto.SummaryRequest{
		PluginName: c.Name,
		StartTime:  startTime,
		EndTime:    endTime,
		Success:    success,
		Error:      errStr,
		Metadata:   metadata,
		Metrics:    metrics,
	}
	resp, err := c.Client.ReportExecutionSummary(ctx, req)
	if err != nil {
		return nil, err
	}

	var execErr error
	if resp.Error != "" {
		execErr = fmt.Errorf(resp.Error)
	}

	return &plugin.ExecutionSummary{
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
func (c *Client) Close() error {
	return c.Conn.Close()
}
