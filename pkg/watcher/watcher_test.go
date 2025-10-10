package watcher_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gregory-chatelier/watchman/pkg/watcher"
)

// TestCommandWatcher_EmptyCommand tests the error handling for an empty command string.
func TestCommandWatcher_EmptyCommand(t *testing.T) {
	cw := watcher.NewCommandWatcher("")

	_, err := cw.Check()
	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}
}

// TestCommandWatcher_Check tests if the CommandWatcher correctly executes a command
// and captures its standard output.
func TestCommandWatcher_Check(t *testing.T) {
	// Use a simple command that prints a known string to stdout
	testCommand := "echo hello world"
	cw := watcher.NewCommandWatcher(testCommand)

	output, err := cw.Check()
	if err != nil {
		t.Fatalf("CommandWatcher.Check failed: %v", err)
	}

	expected := "hello world\n" // echo typically adds a newline
	if string(output) != expected {
		t.Errorf("Expected output %q, got %q", expected, string(output))
	}
}

// TestFileWatcher_Check tests the tail-like behavior of FileWatcher.
func TestFileWatcher_Check(t *testing.T) {
	// 1. Setup: Create a temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.log")

	// Create the file and write initial content
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer f.Close()

	initialContent := "Line 1\n"
	if _, err := f.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}
	f.Sync() // Ensure content is written to disk

	// 2. Test NewFileWatcher: Should start at the end of the file
	fw, err := watcher.NewFileWatcher(filePath)
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	defer fw.Close()

	// First check: Should return nothing, as it starts at the end
	output, err := fw.Check()
	if err != nil {
		t.Fatalf("First Check failed: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("Expected empty output on first check, got %q", string(output))
	}

	// 3. Test reading new content
	newContent := "Line 2\nLine 3\n"
	if _, err := f.WriteString(newContent); err != nil {
		t.Fatalf("Failed to write new content: %v", err)
	}
	f.Sync()

	// Second check: Should return only the new content
	output, err = fw.Check()
	if err != nil {
		t.Fatalf("Second Check failed: %v", err)
	}
	if string(output) != newContent {
		t.Errorf("Expected new content %q, got %q", newContent, string(output))
	}

	// 4. Test reading no new content
	output, err = fw.Check()
	if err != nil {
		t.Fatalf("Third Check failed: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("Expected empty output on third check, got %q", string(output))
	}
}

// TestFileWatcher_FileNotFound tests error handling for non-existent files.
func TestFileWatcher_FileNotFound(t *testing.T) {
	_, err := watcher.NewFileWatcher("/non/existent/path/to/file.log")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestFileWatcher_Close tests closing the file handle.
func TestFileWatcher_Close(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "close.log")
	os.WriteFile(filePath, []byte("test"), 0644)

	fw, err := watcher.NewFileWatcher(filePath)
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}

	if err := fw.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
