package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/gregory-chatelier/watchfor/pkg/executor"
	"github.com/gregory-chatelier/watchfor/pkg/poller"
	"github.com/gregory-chatelier/watchfor/pkg/watcher"
)

var version = "dev" // Default version, will be overwritten by linker

var (
	// Watch Options
	command = pflag.StringP("command", "c", "", "The command to execute and inspect.")
	file    = pflag.StringP("file", "f", "", "The path to the file to read and inspect.")
	pattern = pflag.StringP("pattern", "p", "", "The exact string to search for in the output or file content.")
	regex = pflag.Bool("regex", false, "Enable regex matching for the pattern.")
	ignoreCase = pflag.Bool("ignore-case", false, "Enable case-insensitive matching for the pattern.")

	// Retry Options
	interval    = pflag.Duration("interval", 1*time.Second, "The initial interval between polling attempts (e.g., `5s`, `1m`).")
	maxRetries  = pflag.Int("max-retries", 10, "The maximum number of polling attempts before giving up. `0` means retry forever.")
	backoff     = pflag.Float64("backoff", 1, "The exponential backoff factor. A factor of `1` disables exponential backoff.")
	jitter      = pflag.Float64("jitter", 0, "The jitter factor to apply to the backoff delay (0 to 1). `0` disables jitter.")
	timeout     = pflag.Duration("timeout", 0, "Overall max wait time. Overrides --max-retries. `0` means no timeout.")
	failCommand = pflag.String("on-fail", "", "The command to execute if the pattern is not found.")

	// General Options
	verbose     = pflag.BoolP("verbose", "v", false, "Enable verbose logging.")
	help        = pflag.BoolP("help", "h", false, "Show the help message.")
	showVersion = pflag.BoolP("version", "", false, "Show watchfor version.")
)

func init() {
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] -- [SUCCESS_COMMAND]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Watchfor is a resilient command orchestrator that polls a command or file until a pattern is found.")
		fmt.Fprintln(os.Stderr, "It is designed to replace brittle 'sleep' calls in CI/CD and scripting.")
		fmt.Fprintln(os.Stderr, "Version: " + version)
		fmt.Fprintln(os.Stderr, "Options:")
		pflag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  # Wait for a health check to return 'healthy' with exponential backoff")
		fmt.Fprintln(os.Stderr, "  watchfor -c 'curl -s https://api/health' -p 'status: healthy' --backoff 2 -- ./run_tests.sh")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  # Wait for a log file to contain 'BUILD SUCCESSFUL' for up to 5 minutes")
		fmt.Fprintln(os.Stderr, "  watchfor -f build.log -p 'BUILD SUCCESSFUL' --timeout 5m -- ./deploy.sh")
	}
}

func main() {
	pflag.Parse()

	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("watchfor version %s\n", version)
		os.Exit(0)
	}

	// --- Argument Validation ---
	if *command != "" && *file != "" {
		fmt.Fprintln(os.Stderr, "Error: --command (-c) and --file (-f) cannot be used together.")
		os.Exit(1)
	}
	if *command == "" && *file == "" {
		fmt.Fprintln(os.Stderr, "Error: either --command (-c) or --file (-f) must be specified.")
		os.Exit(1)
	}
	if *pattern == "" {
		fmt.Fprintln(os.Stderr, "Error: --pattern (-p) is required.")
		os.Exit(1)
	}
	if *backoff < 1 {
		fmt.Fprintln(os.Stderr, "Error: --backoff must be >= 1.")
		os.Exit(1)
	}
	if *jitter < 0 || *jitter > 1 {
		fmt.Fprintln(os.Stderr, "Error: --jitter must be between 0 and 1.")
		os.Exit(1)
	}

	// The command to execute on success is all args after '--'
	successCommandArgs := pflag.Args()

	// --- Watcher Selection ---
	var w watcher.Watcher
	var err error

	if *command != "" {
		w = watcher.NewCommandWatcher(*command)
	} else {
		w, err = watcher.NewFileWatcher(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		if fw, ok := w.(*watcher.FileWatcher); ok {
			// Since FileWatcher holds an open file handle, we must ensure it's closed.
			defer fw.Close()
		}
	}

	// --- Run the Poller ---
	poller := poller.New(w, *pattern, *verbose, *regex, *ignoreCase)

	// Create a context for the timeout
	ctx, cancel := context.WithCancel(context.Background())
	if *timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), *timeout)
	}
	defer cancel()

	success := poller.Run(ctx, *interval, *maxRetries, *backoff, *jitter)

	if success {
		fmt.Println("\n✅ Success: Executing success command.")
		successCmdStr := strings.Join(successCommandArgs, " ")
		if err := executor.Execute(successCmdStr); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing success command: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("\n❌ Failure: Executing fail command.")
		if err := executor.Execute(*failCommand); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing fail command: %v\n", err)
			os.Exit(1)
		}
		os.Exit(1) // Exit with a non-zero code on failure
	}
}
