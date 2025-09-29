package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/example/grpc-plugin-app/pkg/shared"
)

// parseParams parses command line arguments in the format key=value into a map
func parseParams(args []string) map[string]string {
	params := make(map[string]string)
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parts[1]
		}
	}
	return params
}

// findAvailablePort finds an available port starting from the given base port
func findAvailablePort(basePort int) int {
	for port := basePort; port < basePort+100; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port
		}
	}
	return basePort // Fallback to base port if no ports are available
}

// displayPluginInfo prints plugin information in a formatted way
func displayPluginInfo(info *shared.PluginInfo, config shared.PluginConfig) {
	fmt.Printf("Plugin Information:\n")
	fmt.Printf("  Name: %s\n", info.Name)
	fmt.Printf("  Version: %s\n", info.Version)
	fmt.Printf("  Description: %s\n", info.Description)
	fmt.Printf("  Type: %s\n", config.Type)
	if config.Type == shared.PluginTypeCommand {
		fmt.Printf("  Command Template: %s\n", config.Command)
	}
	fmt.Printf("  Working Directory: %s\n", config.WorkingDir)
	if len(config.Environment) > 0 {
		fmt.Printf("  Environment Variables:\n")
		for k, v := range config.Environment {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
	fmt.Printf("  Parameters:\n")
	for name, spec := range info.ParameterSchema {
		fmt.Printf("    %s:\n", name)
		fmt.Printf("      Description: %s\n", spec.Description)
		fmt.Printf("      Required: %v\n", spec.Required)
		if spec.DefaultValue != "" {
			fmt.Printf("      Default: %s\n", spec.DefaultValue)
		}
		if configDefault, ok := config.Defaults[name]; ok {
			fmt.Printf("      Config Default: %s\n", configDefault)
		}
		if len(spec.AllowedValues) > 0 {
			fmt.Printf("      Allowed Values: %v\n", spec.AllowedValues)
		}
	}
}

// displayExecutionSummary prints the execution summary in a formatted way
func displayExecutionSummary(summary *shared.ExecutionSummary) {
	log.Printf("Plugin Summary: %s", summary.PluginName)
	log.Printf("  Duration: %.2f ms", summary.Duration)
	log.Printf("  Success: %v", summary.Success)
	if summary.Error != nil {
		log.Printf("  Error: %s", summary.Error.Error())
	}
	log.Printf("  Metadata:")
	for k, v := range summary.Metadata {
		log.Printf("    %s: %s", k, v)
	}
	log.Printf("  Metrics:")
	for k, v := range summary.Metrics {
		log.Printf("    %s: %.2f", k, v)
	}
}

// outputHandler implements shared.OutputHandler for the main application
type outputHandler struct {
	pluginName string
	mutex      sync.Mutex
}

func (h *outputHandler) OnOutput(msg string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	log.Printf("[%s] %s", h.pluginName, msg)
	return nil
}

func (h *outputHandler) OnProgress(p shared.Progress) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	log.Printf("[%s] Progress: %.1f%% (%s - Step %d/%d)",
		h.pluginName, p.PercentComplete, p.Stage, p.CurrentStep, p.TotalSteps)
	return nil
}

func (h *outputHandler) OnError(code, message, details string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if details != "" {
		log.Printf("[%s] Error %s: %s\nDetails: %s", h.pluginName, code, message, details)
	} else {
		log.Printf("[%s] Error %s: %s", h.pluginName, code, message)
	}
	return nil
}

func main() {
	// Set up logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Create a context that will be canceled on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Parse command line flags
	configPath := flag.String("config", "config.json", "Path to configuration file")
	listPlugins := flag.Bool("list", false, "List available plugins")
	showInfo := flag.Bool("info", false, "Show detailed plugin information")
	flag.Parse()

	// Load configuration
	config, err := shared.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Handle -list flag
	if *listPlugins {
		fmt.Println("Available plugins:")
		for _, desc := range config.ListPlugins() {
			fmt.Printf("  %s\n", desc)
		}
		return
	}

	// Get plugin name from arguments
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: plugin-app [-config path/to/config.json] [-list] [-info] <plugin-name> [param1=value1 ...]")
		fmt.Println("Use -list to see available plugins")
		fmt.Println("Use -info to see detailed plugin information")
		os.Exit(1)
	}

	pluginName := args[0]
	pluginConfig, err := config.GetPluginConfig(pluginName)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Validate plugin configuration
	if err := pluginConfig.Validate(); err != nil {
		log.Fatalf("Invalid plugin configuration for %s: %v", pluginName, err)
	}

	// Create plugin manager
	manager := shared.NewPluginManager(config)
	defer manager.StopAll()

	// Start the plugin
	if err := manager.StartPlugin(pluginName, pluginConfig); err != nil {
		log.Fatalf("Failed to start plugin %s: %v", pluginName, err)
	}
	log.Printf("Started plugin: %s (type: %s)", pluginName, pluginConfig.Type)

	// Get the plugin client
	plugin, err := manager.GetPlugin(pluginName)
	if err != nil {
		log.Fatalf("Failed to get plugin %s: %v", pluginName, err)
	}

	// Get plugin info
	info, err := plugin.GetInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get plugin info: %v", err)
	}

	// Handle -info flag
	if *showInfo {
		displayPluginInfo(info, pluginConfig)
		return
	}

	// Parse parameters
	params := parseParams(args[1:])

	// Merge with defaults from plugin schema and config
	for name, spec := range info.ParameterSchema {
		if _, exists := params[name]; !exists {
			// First try config defaults
			if configDefault, ok := pluginConfig.Defaults[name]; ok {
				params[name] = configDefault
			} else if spec.DefaultValue != "" {
				// Fall back to schema defaults
				params[name] = spec.DefaultValue
			}
		}
	}

	// Create output handler
	handler := &outputHandler{
		pluginName: pluginName,
	}

	// Record start time
	startTime := time.Now().UnixNano()

	// Execute plugin
	execErr := plugin.Execute(ctx, params, handler)

	// Record end time
	endTime := time.Now().UnixNano()

	// Prepare metadata and metrics
	metadata := make(map[string]string)
	metrics := make(map[string]float64)

	// Add execution metadata
	metadata["plugin_type"] = string(pluginConfig.Type)
	for k, v := range params {
		metadata[k] = v
	}

	// Add basic metrics
	metrics["execution_time_ms"] = float64(endTime-startTime) / float64(time.Millisecond)

	// Get execution summary
	summary, err := plugin.ReportExecutionSummary(startTime, endTime, execErr == nil, execErr, metadata, metrics)
	if err != nil {
		log.Printf("Failed to get execution summary: %v", err)
	} else {
		displayExecutionSummary(summary)
	}

	// Handle execution error
	if execErr != nil {
		if ctx.Err() == context.Canceled {
			log.Printf("Plugin %s execution canceled", pluginName)
		} else {
			log.Fatalf("Plugin %s execution failed: %v", pluginName, execErr)
		}
	}

	log.Println("Plugin execution completed")
}
