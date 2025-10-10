package executor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Execute runs a command and streams its output to stdout and stderr.
func Execute(command string) error {
	if command == "" {
		return nil // Nothing to do
	}

	fmt.Printf("\n--- Executing: %s ---\n", command)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use cmd /C on Windows
		cmd = exec.Command("cmd", "/C", command)
	} else {
		// Use sh -c on Unix-like systems
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
