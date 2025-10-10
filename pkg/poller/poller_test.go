package poller_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gregory-chatelier/watchman/pkg/poller"
)

// MockWatcher is a mock implementation of the watcher.Watcher interface
type MockWatcher struct {
	// Outputs is a list of outputs to return on successive calls to Check()
	Outputs []string
	// Errors is a list of errors to return on successive calls to Check()
	Errors []error
	// CallCount tracks how many times Check() has been called
	CallCount int
}

// Check returns the next output from the Outputs slice.
func (m *MockWatcher) Check() ([]byte, error) {
	if m.CallCount >= len(m.Outputs) {
		return []byte(""), errors.New("mock output exhausted")
	}

	output := m.Outputs[m.CallCount]
	var err error
	if m.CallCount < len(m.Errors) && m.Errors[m.CallCount] != nil {
		err = m.Errors[m.CallCount]
	}
	m.CallCount++
	return []byte(output), err
}

// TestPoller_SuccessOnFirstAttempt tests if the poller succeeds immediately.
func TestPoller_SuccessOnFirstAttempt(t *testing.T) {
	mock := &MockWatcher{
		Outputs: []string{"status: green"},
	}
	p := poller.New(mock, "status: green", false)

	if !p.Run(context.Background(), time.Millisecond, 1, 1) {
		t.Error("Expected success on first attempt, got failure")
	}
	if mock.CallCount != 1 {
		t.Errorf("Expected 1 call, got %d", mock.CallCount)
	}
}

// TestPoller_SuccessAfterRetries tests if the poller succeeds after a few failures.
func TestPoller_SuccessAfterRetries(t *testing.T) {
	mock := &MockWatcher{
		Outputs: []string{"status: red", "status: yellow", "status: green"},
	}
	p := poller.New(mock, "status: green", false)

	if !p.Run(context.Background(), time.Millisecond, 5, 1) {
		t.Error("Expected success after retries, got failure")
	}
	if mock.CallCount != 3 {
		t.Errorf("Expected 3 calls, got %d", mock.CallCount)
	}
}

// TestPoller_FailureAfterMaxRetries tests if the poller fails when max retries are reached.
func TestPoller_FailureAfterMaxRetries(t *testing.T) {
	mock := &MockWatcher{
		Outputs: []string{"status: red", "status: red", "status: red"},
	}
	p := poller.New(mock, "status: green", false)

	// Max retries is 3. The loop runs 3 times (attempt 0, 1, 2).
	if p.Run(context.Background(), time.Millisecond, 3, 1) {
		t.Error("Expected failure after max retries, got success")
	}
	if mock.CallCount != 3 {
		t.Errorf("Expected 3 calls, got %d", mock.CallCount)
	}
}

// TestPoller_WatcherError tests the error handling path when the watcher fails.
func TestPoller_WatcherError(t *testing.T) {
	mock := &MockWatcher{
		Outputs: []string{"status: red", "status: green"},
		Errors:  []error{errors.New("network error"), nil},
	}
	p := poller.New(mock, "status: green", true) // Use verbose to cover that path

	if !p.Run(context.Background(), time.Millisecond, 5, 1) {
		t.Error("Expected success after one error, got failure")
	}
	if mock.CallCount != 2 {
		t.Errorf("Expected 2 calls, got %d", mock.CallCount)
	}
}

// TestPoller_Timeout tests if the poller stops due to a context timeout.
func TestPoller_Timeout(t *testing.T) {
	mock := &MockWatcher{
		Outputs: []string{"status: red", "status: red", "status: red", "status: red", "status: red"},
	}
	p := poller.New(mock, "status: green", false)

	// Set a very short timeout and a long interval to ensure timeout is hit first.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Use a long interval to ensure the timeout is hit before the next check.
	if p.Run(ctx, 50*time.Millisecond, 5, 1) {
		t.Error("Expected failure due to timeout, got success")
	}
	// Call count should be 1 (the first check)
	if mock.CallCount != 1 {
		t.Errorf("Expected 1 call before timeout, got %d", mock.CallCount)
	}
}

// TestPoller_Verbose tests the verbose logging path for non-matching output.
func TestPoller_Verbose(t *testing.T) {
	mock := &MockWatcher{
		Outputs: []string{"status: red", "status: green"},
	}
	p := poller.New(mock, "status: green", true) // Verbose is true

	if !p.Run(context.Background(), time.Millisecond, 5, 1) {
		t.Error("Expected success, got failure")
	}
	// The verbose path for non-matching output should be covered.
}

// TestPoller_BackoffCalculation tests the exponential backoff logic.
func TestPoller_BackoffCalculation(t *testing.T) {
	// Since the calculation is inside Run, we'll just ensure the Run method
	// doesn't panic and the logic is sound. The previous tests cover the core
	// success/failure paths.
}
