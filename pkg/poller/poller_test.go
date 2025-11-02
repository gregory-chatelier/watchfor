package poller_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gregory-chatelier/watchfor/pkg/poller"
)

// MockWatcher is a mock implementation of the watcher.Watcher interface for testing.
type MockWatcher struct {
	Output   []byte
	Err      error
	Attempts int
}

func (m *MockWatcher) Check() ([]byte, error) {
	m.Attempts++
	return m.Output, m.Err
}

// --- Poller Tests ---

func TestPoller_Run_MatchingLogic(t *testing.T) {
	testCases := []struct {
		name       string
		pattern    string
		output     string
		regex      bool
		ignoreCase bool
		expected   bool
	}{
		{"Simple Match", "SUCCESS", "output with SUCCESS", false, false, true},
		{"Simple Match Fail", "FAIL", "output with SUCCESS", false, false, false},
		{"Ignore Case Match", "success", "output with SUCCESS", false, true, true},
		{"Ignore Case Match Fail", "fail", "output with SUCCESS", false, true, false},
		{"Regex Match", "S.CCESS", "output with SUCCESS", true, false, true},
		{"Regex Match Fail", "F.IL", "output with SUCCESS", true, false, false},
		{"Regex Ignore Case Match", "s.ccess", "output with SUCCESS", true, true, true},
		{"Regex Ignore Case Match Fail", "f.il", "output with SUCCESS", true, true, false},
		{"Invalid Regex", "[a-z", "any output", true, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockWatcher := &MockWatcher{Output: []byte(tc.output)}
			p := poller.New(mockWatcher, tc.pattern, false, tc.regex, tc.ignoreCase)

			success := p.Run(context.Background(), 1*time.Millisecond, 1, 1)

			if success != tc.expected {
				t.Errorf("Expected success=%v, but got %v", tc.expected, success)
			}
		})
	}
}

func TestPoller_Run_Success(t *testing.T) {
	mockWatcher := &MockWatcher{
		Output: []byte("some log output"),
	}
	p := poller.New(mockWatcher, "SUCCESS", false, false, false)

	// The mock watcher for this test needs to change its output
	go func() {
		time.Sleep(2 * time.Millisecond)
		mockWatcher.Output = []byte("SUCCESS")
	}()

	// Run with enough retries to succeed on the 3rd attempt
	success := p.Run(context.Background(), 1*time.Millisecond, 5, 1)

	if !success {
		t.Errorf("Expected Run to succeed, but it failed")
	}
}

func TestPoller_Run_MaxRetries(t *testing.T) {
	mockWatcher := &MockWatcher{
		Output: []byte("some log output"),
	}
	p := poller.New(mockWatcher, "SUCCESS", false, false, false)

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
		Output: []byte("some log output"),
	}
	p := poller.New(mockWatcher, "SUCCESS", false, false, false)

	// Set a very short timeout that will expire before the 3rd attempt
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	// Use a long interval to ensure the timeout is hit during the wait
	success := p.Run(ctx, 100*time.Millisecond, 10, 1)

	if success {
		t.Errorf("Expected Run to fail due to timeout, but it succeeded")
	}
}

func TestPoller_Run_WatcherError(t *testing.T) {
	mockWatcher := &MockWatcher{
		Err: errors.New("simulated watcher error"),
	}
	p := poller.New(mockWatcher, "SUCCESS", true, false, false) // Verbose to ensure logging path is hit

	success := p.Run(context.Background(), 1*time.Millisecond, 2, 1)

	if success {
		t.Errorf("Expected Run to fail due to watcher error, but it succeeded")
	}
}

