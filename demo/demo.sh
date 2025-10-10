#!/bin/bash

# --- Watchman Demo Script ---
# This script demonstrates the two main modes of the watchman utility:
# 1. Command Mode: Polling a flaky health check with retries and backoff.
# 2. File Mode: Watching a log file for a specific success message.

set -e

# --- Configuration ---
LOG_FILE="demo.log"
MOCK_SCRIPT="./demo/mock_health_check.sh"
WATCHMAN_BIN="./watchman.exe" # Adjusted path for Windows

# Explicitly clean up mock file before starting
rm -f "/tmp/watchman_mock_count"

# --- Helper Functions ---

cleanup() {
    echo -e "\n--- Cleaning up temporary files ---"
    rm -f "$LOG_FILE" "/tmp/watchman_mock_count"
}

trap cleanup EXIT

# Make scripts executable
chmod +x demo/mock_health_check.sh

echo "--- Starting Watchman Demo ---"

# ----------------------------------------------------------------------
# DEMO 1: Command Mode (Resilient Health Check)
# ----------------------------------------------------------------------
echo -e "\n[DEMO 1: Command Mode]"
echo "Simulating a flaky service that takes 3 attempts to become 'green'."
echo "watchman will poll every 100ms with a backoff factor of 2."

"$WATCHMAN_BIN" \
    --command "$MOCK_SCRIPT" \
    --pattern "status: green" \
    --interval 100ms \
    --max-retries 5 \
    --backoff 2 \
    --verbose \
    --on-fail "echo 'Demo 1 FAILED: Service never became green.'" \
    -- echo "Demo 1 SUCCESS: Service is now healthy!"

# ----------------------------------------------------------------------
# DEMO 2: File Mode (Waiting for Log Message)
# ----------------------------------------------------------------------
echo -e "\n[DEMO 2: File Mode]"
echo "Simulating a long-running build process writing to $LOG_FILE."
echo "watchman will wait for 'BUILD SUCCESSFUL' before running the deploy command."

# 1. Start a background process to write to the log file
echo "Starting background process to write log messages..."
(
    echo "Build started..." > "$LOG_FILE"
    sleep 1
    echo "Compiling module A..." >> "$LOG_FILE"
    sleep 1
    echo "Compiling module B..." >> "$LOG_FILE"
    sleep 2
    echo "BUILD SUCCESSFUL" >> "$LOG_FILE"
) & LOG_WRITER_PID=$!

# 2. Start watchman to monitor the log file
echo "Starting watchman to monitor $LOG_FILE..."
"$WATCHMAN_BIN" \
    --file "$LOG_FILE" \
    --pattern "BUILD SUCCESSFUL" \
    --interval 500ms \
    --timeout 10s \
    --verbose \
    --on-fail "echo 'Demo 2 FAILED: Build timed out or failed.'" \
    -- echo "Demo 2 SUCCESS: Deploying application..."

# Wait for the background process to finish
wait $LOG_WRITER_PID 2>/dev/null

# ----------------------------------------------------------------------
# DEMO 3: Command Mode (Failure Scenario)
# ----------------------------------------------------------------------
echo -e "\n[DEMO 3: Command Mode - Failure]"
echo "Simulating a service that never becomes healthy. watchman should fail after 3 retries."

(
    set +e # Temporarily disable exit on error for this subshell
    "$WATCHMAN_BIN" \
        --command "echo 'status: red'" \
        --pattern "status: green" \
        --interval 50ms \
        --max-retries 3 \
        --verbose \
        --on-fail "echo 'Demo 3 SUCCESS: watchman correctly executed the on-fail command.'" \
        -- echo "Demo 3 FAILED: Success command should NOT run."
    
    WATCHMAN_EXIT_CODE=$?
    if [ $WATCHMAN_EXIT_CODE -eq 1 ]; then
        echo "Demo 3 Verification: watchman exited with expected failure code 1."
    else
        echo "Demo 3 Verification FAILED: watchman exited with unexpected code $WATCHMAN_EXIT_CODE."
        exit 1 # Exit the subshell with error if verification fails
    fi
)

echo -e "\n--- Demo Complete ---"
