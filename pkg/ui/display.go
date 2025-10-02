package ui

import (
	"fmt"
	"log"
	"strings"

	"github.com/example/grpc-plugin-app/pkg/plugin"
)

// displayPluginInfo prints plugin information in a formatted way
func DisplayPluginInfo(info *plugin.PluginInfo, config plugin.PluginConfig) {
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
	if config.Type == plugin.PluginTypeCommand {
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
func DisplayExecutionSummary(summary *plugin.ExecutionSummary) {
	log.Printf("Plugin Summary: %s", summary.PluginName)
	log.Printf("  Success: %v", summary.Success)
	log.Printf("  Duration: %.2fms", summary.Duration)
	if summary.Error != nil {
		log.Printf("  Error: %v", summary.Error)
	}
	if len(summary.Metadata) > 0 {
		log.Printf("  Metadata:")
		for k, v := range summary.Metadata {
			log.Printf("    %s: %s", k, v)
		}
	}
	if len(summary.Metrics) > 0 {
		log.Printf("  Metrics:")
		for k, v := range summary.Metrics {
			log.Printf("    %s: %.2f", k, v)
		}
	}
}
