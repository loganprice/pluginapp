package main

import (
	"fmt"
	"log"

	"github.com/example/grpc-plugin-app/cmd"
	"github.com/example/grpc-plugin-app/internal/manager"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "A plugin-based application framework",
	Long:  `A CLI application that manages and executes plugins using gRPC.`,
	PersistentPreRunE: func(cobraCmd *cobra.Command, args []string) error {
		// Load configuration
		var err error
		cmd.Config, err = manager.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.json", "config file (default is config.json)")
	rootCmd.AddCommand(cmd.NewListCmd())
	rootCmd.AddCommand(cmd.NewInfoCmd())
	rootCmd.AddCommand(cmd.NewRunCmd())
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
