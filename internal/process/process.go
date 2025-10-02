
package process

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/example/grpc-plugin-app/pkg/plugin"
)

// StartPluginFromConfig starts a plugin using the shared configuration
func StartPluginFromConfig(config plugin.PluginConfig) (*exec.Cmd, error) {
	// Start the plugin process
	cmd := exec.Command(config.Path, "-port", fmt.Sprintf("%d", config.Port))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start plugin: %v", err)
	}

	return cmd, nil
}

// StopPlugin stops a running plugin process
func StopPlugin(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("plugin process not found")
	}
	return cmd.Process.Kill()
}
