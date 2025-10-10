package executor_test

import (
	"os"
	"testing"

	"github.com/gregory-chatelier/watchfor/pkg/executor"
)

// TestExecute_Success tests running a simple, successful command.
func TestExecute_Success(t *testing.T) {
	// Use a command that is guaranteed to succeed and print something
	var cmd string
	if os.Getenv("GOOS") == "windows" {
		cmd = "cmd /C echo success"
	} else {
		cmd = "echo success"
	}

	err := executor.Execute(cmd)
	if err != nil {
		t.Errorf("Expected command to succeed, but got error: %v", err)
	}
}

// TestExecute_Failure tests running a command that is guaranteed to fail.
func TestExecute_Failure(t *testing.T) {
	// Use a command that is guaranteed to fail (e.g., a non-existent command)
	// or a command that returns a non-zero exit code.
	var cmd string
	if os.Getenv("GOOS") == "windows" {
		// On Windows, 'cmd /C exit 1' is a reliable way to force a non-zero exit code
		cmd = "cmd /C exit 1"
	} else {
		// On Unix-like systems, 'false' returns exit code 1
		cmd = "false"
	}

	err := executor.Execute(cmd)
	if err == nil {
		t.Error("Expected command to fail (non-zero exit code), but got nil error")
	}
}

// TestExecute_EmptyCommand tests running an empty command string.
func TestExecute_EmptyCommand(t *testing.T) {
	err := executor.Execute("")
	if err != nil {
		t.Errorf("Expected nil error for empty command, got: %v", err)
	}
}
