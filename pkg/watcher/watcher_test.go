package watcher_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/gregory-chatelier/watchfor/pkg/watcher"
)

// Helper function to create a temporary file with content
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "watchfor-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if content != "" {
		if _, err := tmpfile.WriteString(content); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}
	tmpfile.Close()
	return tmpfile.Name()
}

// --- CommandWatcher Tests ---

func TestCommandWatcher_Check_Success(t *testing.T) {
	// Command that succeeds and prints output
	cmdStr := "echo hello world"
	if runtime.GOOS == "windows" {
		cmdStr = "echo hello world" // cmd /C echo does not need quotes
	}
	cw := watcher.NewCommandWatcher(cmdStr)

	output, err := cw.Check()

	if err != nil {
		t.Fatalf("CommandWatcher failed with error: %v", err)
	}
	// Normalize newlines to accommodate OS differences
	normalizedOutput := strings.ReplaceAll(string(output), "\r\n", " ")
	normalizedOutput = strings.ReplaceAll(normalizedOutput, "\n", " ")
	
	if !strings.Contains(normalizedOutput, "hello world") {
		t.Errorf("Expected output to contain 'hello world', got: %s", string(output))
	}
}

func TestCommandWatcher_Check_NonZeroExit(t *testing.T) {
	// Command that fails (non-zero exit code) but still prints output
	cmdStr := "sh -c 'echo error output; exit 1'"
	if runtime.GOOS == "windows" {
		// Windows equivalent: echo output, then exit 1
		cmdStr = "echo error output & exit 1"
	}
	cw := watcher.NewCommandWatcher(cmdStr)

	output, err := cw.Check()

	if err == nil {
		t.Fatalf("Expected CommandWatcher to return an error for non-zero exit code, got nil")
	}
	if !strings.Contains(string(output), "error output") {
		t.Errorf("Expected output to contain 'error output', got: %s", string(output))
	}
}

// --- FileWatcher Tests ---

func TestFileWatcher_Check_Append(t *testing.T) {
	filePath := createTempFile(t, "initial content\n")
	defer os.Remove(filePath)

	fw, err := watcher.NewFileWatcher(filePath)
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// 1. Initial check should return nothing (starts at EOF)
	output, err := fw.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("Expected initial check to return 0 bytes, got %d: %s", len(output), string(output))
	}

	// 2. Append new content
	f, _ := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("new line 1\n")
	f.Close()

	// 3. Check again, should return new content
	output, err = fw.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	expected := "new line 1\n"
	if string(output) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(output))
	}
}

func TestFileWatcher_Check_Truncation(t *testing.T) {
	filePath := createTempFile(t, "1234567890\n") // 11 bytes
	defer os.Remove(filePath)

	fw, err := watcher.NewFileWatcher(filePath)
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// Read once to set offset to EOF (11)
	fw.Check()

	// 1. Truncate the file to 0 bytes (simulating logrotate)
	f, _ := os.OpenFile(filePath, os.O_TRUNC|os.O_WRONLY, 0644)
	f.Close() // File size is now 0.

	// 2. Call Check() to trigger the offset reset (11 > 0 -> offset = 0)
	// This check should return 0 bytes.
	output, err := fw.Check()
	if err != nil {
		t.Fatalf("Check failed after truncation: %v", err)
	}
	if len(output) != 0 {
		t.Fatalf("Expected 0 bytes after truncation, got: %s", string(output))
	}

	// 3. Append new content
	f, _ = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("new content after truncate\n")
	f.Close()

	// 4. Check again, offset should be 0 and returned the new content
	output, err = fw.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	expected := "new content after truncate\n"
	if string(output) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(output))
	}
}

