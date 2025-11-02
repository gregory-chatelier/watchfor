package poller

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/gregory-chatelier/watchfor/pkg/watcher"
)

// Poller manages the polling loop, checking for a pattern from a watcher.
type Poller struct {
	w          watcher.Watcher
	pattern    string
	verbose    bool
	regex      bool
	ignoreCase bool
}

// New creates a new Poller.
func New(w watcher.Watcher, pattern string, verbose bool, regex bool, ignoreCase bool) *Poller {
	return &Poller{
		w:          w,
		pattern:    pattern,
		verbose:    verbose,
		regex:      regex,
		ignoreCase: ignoreCase,
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
				// Print the output even on error, as the pattern might be in the combined output
				if len(output) > 0 {
					fmt.Printf("Attempt %d: Output:\n%s\n", attempt+1, string(output))
				}
			}
		} else if p.verbose {
			fmt.Printf("Attempt %d: Command successful. Checking output...\n", attempt+1)
			if len(output) > 0 {
				fmt.Printf("Attempt %d: Output:\n%s\n", attempt+1, string(output))
			}
		}

		matched, err := p.match(output)
		if err != nil {
			fmt.Printf("Error matching pattern: %v\n", err)
			return false // Consider this a fatal error
		}

		if matched {
			fmt.Println("Pattern found!")
			return true // Success
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

func (p *Poller) match(output []byte) (bool, error) {
	if p.regex {
		pattern := p.pattern
		if p.ignoreCase {
			pattern = "(?i)" + pattern
		}
		return regexp.Match(pattern, output)
	}

	if p.ignoreCase {
		return bytes.Contains(bytes.ToLower(output), bytes.ToLower([]byte(p.pattern))), nil
	}

	return bytes.Contains(output, []byte(p.pattern)), nil
}
