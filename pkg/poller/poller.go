package poller

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gregory-chatelier/watchman/pkg/watcher"
)

// Poller manages the polling loop, checking for a pattern from a watcher.
type Poller struct {
	w       watcher.Watcher
	pattern []byte
	verbose bool
}

// New creates a new Poller.
func New(w watcher.Watcher, pattern string, verbose bool) *Poller {
	return &Poller{
		w:       w,
		pattern: []byte(pattern),
		verbose: verbose,
	}
}

// Run starts the polling loop and returns true if the pattern is found.
func (p *Poller) Run(ctx context.Context, interval time.Duration, maxRetries int, backoff float64) bool {
	attempt := 0
	for {
		output, err := p.w.Check()
		if err != nil {
			if p.verbose {
				fmt.Printf("Attempt %d: Error checking watcher: %v\n", attempt+1, err)
			}
		} else {
			if p.verbose {
				fmt.Printf("Attempt %d: Checking output...\n", attempt+1)
			}
			if bytes.Contains(output, p.pattern) {
				fmt.Println("Pattern found!")
				return true // Success
			}
		}

		// Check if we should stop.
		if maxRetries > 0 && attempt >= maxRetries-1 {
			fmt.Println("Max retries reached.")
			return false // Failure
		}

		attempt++

		// Calculate next delay
		delay := float64(interval) * math.Pow(backoff, float64(attempt))

		// Cap the delay to prevent overflow and excessive waiting (e.g., 1 hour max)
		maxDelay := float64(time.Hour)
		if delay > maxDelay {
			delay = maxDelay
		}

		nextInterval := time.Duration(delay)

		if p.verbose {
			fmt.Printf("No pattern match. Waiting %s before next attempt.\n", nextInterval)
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			fmt.Println("Timeout reached.")
			return false // Failure due to timeout
		case <-time.After(nextInterval):
			// Continue to next iteration
		}
	}
}
