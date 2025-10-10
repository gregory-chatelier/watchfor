package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Execute runs a command and streams its output to stdout and stderr.
func Execute(command string) error {
	if command == "" {
		return nil // Nothing to do
	}

	fmt.Printf("\n--- Executing: %s ---\n", command)

	// This is a simple approach. For production, a more robust shell-parsing library might be needed.
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
