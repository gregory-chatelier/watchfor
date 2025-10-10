package poller_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gregory-chatelier/watchman/pkg/poller"
)

// MockWatcher is a mock implementation of the watcher.Watcher interface for testing.
type MockWatcher struct {
	Attempts int
	Pattern  string
	Output   []byte
	Err      error
}

func (m *MockWatcher) Check() ([]byte, error) {
	m.Attempts++
	if m.Attempts == 3 {
		return []byte(m.Pattern), nil // Succeeds on the 3rd attempt
	}
	if m.Err != nil {
		return m.Output, m.Err
	}
	return m.Output, nil
}

// --- Poller Tests ---

func TestPoller_Run_Success(t *testing.T) {
	mockWatcher := &MockWatcher{
		Pattern: "SUCCESS",
		Output:  []byte("some log output"),
	}
	p := poller.New(mockWatcher, "SUCCESS", false)

	// Run with enough retries to succeed on the 3rd attempt
	success := p.Run(context.Background(), 1*time.Millisecond, 5, 1)

	if !success {
		t.Errorf("Expected Run to succeed, but it failed")
	}
	if mockWatcher.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", mockWatcher.Attempts)
	}
}

func TestPoller_Run_MaxRetries(t *testing.T) {
	mockWatcher := &MockWatcher{
		Pattern: "NEVER_FOUND",
		Output:  []byte("some log output"),
	}
	p := poller.New(mockWatcher, "SUCCESS", false)

	// Run with only 2 retries (will fail)
	success := p.Run(context.Background(), 1*time.Millisecond, 2, 1)

	if success {
		t.Errorf("Expected Run to fail due to max retries, but it succeeded")
	}
	if mockWatcher.Attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", mockWatcher.Attempts)
	}
}

func TestPoller_Run_Timeout(t *testing.T) {
	mockWatcher := &MockWatcher{
		Pattern: "SUCCESS",
		Output:  []byte("some log output"),
	}
	p := poller.New(mockWatcher, "SUCCESS", false)

	// Set a very short timeout that will expire before the 3rd attempt
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	// Use a long interval to ensure the timeout is hit during the wait
	success := p.Run(ctx, 100*time.Millisecond, 10, 1)

	if success {
		t.Errorf("Expected Run to fail due to timeout, but it succeeded")
	}
	// Attempts should be 1 or 2, depending on timing, but definitely less than 3
	if mockWatcher.Attempts >= 3 {
		t.Errorf("Expected less than 3 attempts due to timeout, got %d", mockWatcher.Attempts)
	}
}

func TestPoller_Run_Backoff(t *testing.T) {
	mockWatcher := &MockWatcher{
		Pattern: "SUCCESS",
		Output:  []byte("some log output"),
	}
	p := poller.New(mockWatcher, "SUCCESS", false)

	// Measure the time taken for 3 attempts with backoff=2 and interval=10ms
	start := time.Now()
	p.Run(context.Background(), 10*time.Millisecond, 3, 2)
	duration := time.Since(start)

	// Expected delays:
	// Attempt 1: 0ms (no wait before first check)
	// Wait 1: 10ms * 2^1 = 20ms
	// Wait 2: 10ms * 2^2 = 40ms
	// Total expected wait time: 60ms.
	// We add a buffer for execution time.
	expectedMinDuration := 60 * time.Millisecond
	if duration < expectedMinDuration {
		t.Errorf("Expected duration to be at least %s, got %s", expectedMinDuration, duration)
	}
}

func TestPoller_Run_WatcherError(t *testing.T) {
	mockWatcher := &MockWatcher{
		Pattern: "SUCCESS",
		Output:  []byte("some error output"),
		Err:     errors.New("simulated watcher error"),
	}
	p := poller.New(mockWatcher, "SUCCESS", true) // Verbose to ensure logging path is hit

	// Run with enough retries to succeed on the 3rd attempt
	success := p.Run(context.Background(), 1*time.Millisecond, 5, 1)

	if !success {
		t.Errorf("Expected Run to succeed, but it failed")
	}
	if mockWatcher.Attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", mockWatcher.Attempts)
	}
}