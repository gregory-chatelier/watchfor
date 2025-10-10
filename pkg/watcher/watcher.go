package watcher

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

// Watcher defines the interface for checking a source for a pattern.
type Watcher interface {
	// Check reads the source and returns the content.
	Check() ([]byte, error)
}

// --- Command Watcher ---

// CommandWatcher runs a command and captures its output.
type CommandWatcher struct {
	command string
}

// NewCommandWatcher creates a new watcher for a shell command.
func NewCommandWatcher(cmd string) *CommandWatcher {
	return &CommandWatcher{command: cmd}
}

// Check executes the command and returns its standard output.
func (cw *CommandWatcher) Check() ([]byte, error) {
	// Execute the command using a shell interpreter (sh -c) to handle shell scripts,
	// pipes, and complex commands correctly.
	cmd := exec.Command("sh", "-c", cw.command)

	// cmd.Output() returns the combined stdout and stderr if the command exits with a non-zero status.
	// We only care about stdout for pattern matching.
	// However, for simplicity and to capture all output for pattern matching, we use cmd.Output().
	// If the command fails (non-zero exit code), cmd.Output() returns an error, but the output is still available in the error.
	// We will return the output and ignore the error for now, as the pattern check is the primary concern.
	// A better approach is to use cmd.CombinedOutput() or handle the error to extract the output.
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the command fails (non-zero exit code), we still return the output
		// so the pattern can be checked against the error message/output.
		return output, nil
	}

	return output, nil
}

// --- File Watcher ---

// FileWatcher reads new content from a file, mimicking `tail -f`.
type FileWatcher struct {
	filepath string
	file     *os.File
	offset   int64
}

// NewFileWatcher creates a new watcher for a file path.
func NewFileWatcher(path string) (*FileWatcher, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// Start reading from the end of the file.
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		return nil, err
	}

	return &FileWatcher{
		filepath: path,
		file:     file,
		offset:   offset,
	}, nil
}

// Check reads any new content appended to the file since the last check.
func (fw *FileWatcher) Check() ([]byte, error) {
	// Get current file info to check for truncation
	info, err := fw.file.Stat()
	if err != nil {
		return nil, err
	}

	// Check for truncation: if the current offset is greater than the file size,
	// the file has been truncated (e.g., by logrotate). Reset offset to 0.
	if fw.offset > info.Size() {
		fw.offset = 0
	}

	// Move the cursor to the last known offset.
	_, err = fw.file.Seek(fw.offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Read all new content from the current offset to the end.
	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, fw.file)
	if err != nil {
		return nil, err
	}

	// Update the offset for the next read.
	fw.offset += n

	return buf.Bytes(), nil
}

// Close closes the file handle.
func (fw *FileWatcher) Close() error {
	if fw.file != nil {
		return fw.file.Close()
	}
	return nil
}
