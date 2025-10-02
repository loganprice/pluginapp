package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available plugins:")
			for _, desc := range Config.ListPlugins() {
				fmt.Printf("  %s\n", desc)
			}
		},
	}
}
