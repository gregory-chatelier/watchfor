package watcher

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"runtime"
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
	var cmd *exec.Cmd
	var shell, flag string

	if runtime.GOOS == "windows" {
		shell = "powershell"
		flag = "-Command"
	} else {
		shell = "sh"
		flag = "-c"
	}

	cmd = exec.Command(shell, flag, cw.command)

	// Use CombinedOutput to capture both stdout and stderr for pattern matching
	output, err := cmd.CombinedOutput()

	// Return the output and the error (if any).
	// The poller will decide whether to treat a non-zero exit code as a failure.
	return output, err
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
		err := fw.file.Close()
		fw.file = nil // Prevent double close
		return err
	}
	return nil
}