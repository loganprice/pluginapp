package cmd

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/example/grpc-plugin-app/internal/app"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [plugin-name] [param1=value1 ...]",
		Short: "Run a specific plugin",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]

			// Check for a help flag in the arguments
			for _, arg := range args[1:] {
				if arg == "--help" || arg == "-h" {
					return app.ShowPluginInfo(Config, pluginName)
				}
			}

			pluginParams := app.ParsePluginFlags(args[1:])

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			return app.ExecutePlugin(ctx, Config, pluginName, pluginParams)
		},
	}
}
