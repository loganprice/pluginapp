
package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/example/grpc-plugin-app/internal/manager"
	"github.com/example/grpc-plugin-app/pkg/ui"
)

func ShowPluginInfo(config *manager.AppConfig, pluginName string) error {
	pluginConfig, err := config.GetPluginConfig(pluginName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pluginManager := manager.NewPluginManager(config)
	defer pluginManager.StopAll()

	if err := pluginManager.StartPlugin(pluginName, pluginConfig, make(map[string]string)); err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", pluginName, err)
	}

	p, err := pluginManager.GetPlugin(pluginName)
	if err != nil {
		return fmt.Errorf("failed to get plugin %s: %w", pluginName, err)
	}

	info, err := p.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get plugin info: %w", err)
	}

	ui.DisplayPluginInfo(info, pluginConfig)
	return nil
}

func ExecutePlugin(ctx context.Context, config *manager.AppConfig, pluginName string, params map[string]string) error {
	pluginConfig, err := config.GetPluginConfig(pluginName)
	if err != nil {
		return err
	}

	if err := pluginConfig.Validate(); err != nil {
		return fmt.Errorf("invalid plugin configuration for %s: %w", pluginName, err)
	}

	pluginManager := manager.NewPluginManager(config)
	defer pluginManager.StopAll()

	if err := pluginManager.StartPlugin(pluginName, pluginConfig, params); err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", pluginName, err)
	}
	log.Printf("Started plugin: %s (type: %s)", pluginName, pluginConfig.Type)

	p, err := pluginManager.GetPlugin(pluginName)
	if err != nil {
		return fmt.Errorf("failed to get plugin %s: %w", pluginName, err)
	}

	info, err := p.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get plugin info: %w", err)
	}

	// Merge params with defaults
	for name, spec := range info.ParameterSchema {
		if _, exists := params[name]; !exists {
			if configDefault, ok := pluginConfig.Defaults[name]; ok {
				params[name] = configDefault
			} else if spec.DefaultValue != "" {
				params[name] = spec.DefaultValue
			}
		}
	}

	handler := ui.NewOutputHandler(pluginName)
	startTime := time.Now().UnixNano()

	execErr := p.Execute(ctx, params, handler)

	endTime := time.Now().UnixNano()

	metadata := make(map[string]string)
	metrics := make(map[string]float64)
	metadata["plugin_type"] = string(pluginConfig.Type)
	for k, v := range params {
		metadata[k] = v
	}
	metrics["execution_time_ms"] = float64(endTime-startTime) / float64(time.Millisecond)

	summary, err := p.ReportExecutionSummary(startTime, endTime, execErr == nil, execErr, metadata, metrics)
	if err != nil {
		log.Printf("Failed to get execution summary: %v", err)
	} else {
		ui.DisplayExecutionSummary(summary)
	}

	if execErr != nil {
		if ctx.Err() == context.Canceled {
			log.Printf("Plugin %s execution canceled", pluginName)
			return nil // Not a fatal error
		}
		return fmt.Errorf("plugin %s execution failed: %w", pluginName, execErr)
	}

	log.Println("Plugin execution completed successfully")
	return nil
}

// ParsePluginFlags parses command line arguments into a map. It supports:
// --key=value
// --key value
// --key (as a boolean true)
func ParsePluginFlags(args []string) map[string]string {
	params := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			continue // Ignore non-flag arguments
		}

		key := strings.TrimLeft(arg, "-")

		// Handle --key=value
		if strings.Contains(key, "=") {
			parts := strings.SplitN(key, "=", 2)
			params[parts[0]] = parts[1]
			continue
		}

		// Handle --key value
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			params[key] = args[i+1]
			i++ // Skip the next arg as it was the value
			continue
		}

		// Handle boolean flag like --verbose
		params[key] = "true"
	}
	return params
}
