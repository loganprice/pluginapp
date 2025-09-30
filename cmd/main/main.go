package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/example/grpc-plugin-app/pkg/shared"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	config  *shared.AppConfig
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "A plugin-based application framework",
	Long:  `A CLI application that manages and executes plugins using gRPC.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		var err error
		config, err = shared.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available plugins",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available plugins:")
		for _, desc := range config.ListPlugins() {
			fmt.Printf("  %s\n", desc)
		}
	},
}

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info [plugin-name]",
	Short: "Show detailed information for a specific plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return showPluginHelp(args[0])
	},
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [plugin-name] [param1=value1 ...]",
	Short: "Run a specific plugin",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]

		// Check for a help flag in the arguments
		for _, arg := range args[1:] {
			if arg == "--help" || arg == "-h" {
				return showPluginHelp(pluginName)
			}
		}

		pluginParams := parsePluginFlags(args[1:])

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		return executePlugin(ctx, pluginName, pluginParams)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.json", "config file (default is config.json)")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(runCmd)

	// Stop parsing flags after the first non-flag argument (the plugin name)
	runCmd.Flags().SetInterspersed(false)
}

func showPluginHelp(pluginName string) error {
	pluginConfig, err := config.GetPluginConfig(pluginName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	manager := shared.NewPluginManager(config)
	defer manager.StopAll()

	if err := manager.StartPlugin(pluginName, pluginConfig); err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", pluginName, err)
	}

	plugin, err := manager.GetPlugin(pluginName)
	if err != nil {
		return fmt.Errorf("failed to get plugin %s: %w", pluginName, err)
	}

	info, err := plugin.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get plugin info: %w", err)
	}

	displayPluginInfo(info, pluginConfig)
	return nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func executePlugin(ctx context.Context, pluginName string, params map[string]string) error {
	pluginConfig, err := config.GetPluginConfig(pluginName)
	if err != nil {
		return err
	}

	if err := pluginConfig.Validate(); err != nil {
		return fmt.Errorf("invalid plugin configuration for %s: %w", pluginName, err)
	}

	manager := shared.NewPluginManager(config)
	defer manager.StopAll()

	if err := manager.StartPlugin(pluginName, pluginConfig); err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", pluginName, err)
	}
	log.Printf("Started plugin: %s (type: %s)", pluginName, pluginConfig.Type)

	plugin, err := manager.GetPlugin(pluginName)
	if err != nil {
		return fmt.Errorf("failed to get plugin %s: %w", pluginName, err)
	}

	info, err := plugin.GetInfo(ctx)
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

	handler := &outputHandler{pluginName: pluginName}
	startTime := time.Now().UnixNano()

	execErr := plugin.Execute(ctx, params, handler)

	endTime := time.Now().UnixNano()

	metadata := make(map[string]string)
	metrics := make(map[string]float64)
	metadata["plugin_type"] = string(pluginConfig.Type)
	for k, v := range params {
		metadata[k] = v
	}
	metrics["execution_time_ms"] = float64(endTime-startTime) / float64(time.Millisecond)

	summary, err := plugin.ReportExecutionSummary(startTime, endTime, execErr == nil, execErr, metadata, metrics)
	if err != nil {
		log.Printf("Failed to get execution summary: %v", err)
	} else {
		displayExecutionSummary(summary)
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

// outputHandler, display functions, and parseParams remain the same

// parsePluginFlags parses command line arguments into a map. It supports:
// --key=value
// --key value
// --key (as a boolean true)
func parsePluginFlags(args []string) map[string]string {
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

// displayPluginInfo prints plugin information in a formatted way
func displayPluginInfo(info *shared.PluginInfo, config shared.PluginConfig) {
	fmt.Printf("Plugin Information:\n")
	fmt.Printf("  Name: %s\n", info.Name)
	fmt.Printf("  Version: %s\n", info.Version)
	fmt.Printf("  Description: %s\n", info.Description)
	fmt.Printf("  Type: %s\n", config.Type)

	// Build usage string
	var usageParams []string
	for name, spec := range info.ParameterSchema {
		if spec.Required {
			usageParams = append(usageParams, fmt.Sprintf("--%s <value>", name))
		} else {
			usageParams = append(usageParams, fmt.Sprintf("[--%s <value>]", name))
		}
	}
	fmt.Printf("\nUsage:\n")
	fmt.Printf("  app run %s %s\n\n", info.Name, strings.Join(usageParams, " "))

	fmt.Printf("Details:\n")
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
	fmt.Printf("\nParameters:\n")
	for name, spec := range info.ParameterSchema {
		fmt.Printf("  - %s:\n", name)
		fmt.Printf("      Description: %s\n", spec.Description)
		fmt.Printf("      Required: %v\n", spec.Required)
		if spec.DefaultValue != "" {
			fmt.Printf("      Schema Default: %s\n", spec.DefaultValue)
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
	// ... (rest of the function is the same)
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