package cmd

import (
	"github.com/example/grpc-plugin-app/internal/app"
	"github.com/spf13/cobra"
)

func NewInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [plugin-name]",
		Short: "Show detailed information for a specific plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.ShowPluginInfo(Config, args[0])
		},
	}
}
