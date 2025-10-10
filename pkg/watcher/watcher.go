package watcher

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
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
	// Use "sh -c" or "cmd /C" to properly handle commands with arguments.
	// This is a simplified approach. A more robust solution would parse the command and args.
	var cmd *exec.Cmd
	parts := strings.Fields(cw.command)
	if len(parts) == 0 {
		return nil, os.ErrInvalid
	}
	cmd = exec.Command(parts[0], parts[1:]...)

	return cmd.Output()
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
	// Move the cursor to the last known offset.
	_, err := fw.file.Seek(fw.offset, io.SeekStart)
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
