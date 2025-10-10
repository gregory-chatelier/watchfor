# watchman

**watchman** is a resilient command orchestrator written in Go. It is designed to replace brittle `sleep` calls in CI/CD pipelines and scripts by waiting for a specific *state* (a pattern in a command's output or a file's content) before proceeding.

It features:
*   **Command Polling:** Repeatedly runs a command (e.g., a health check) and inspects its output.
*   **File Watching:** Efficiently monitors a file (like a log) for new content, similar to `tail -f`.
*   **Resilience:** Built-in support for maximum retries, exponential backoff, and a global timeout.
*   **Graceful Failure:** Executes a fallback command (`--on-fail`) if the condition is never met.

## Installation

To install `watchman`, ensure you have Go installed and run:

```bash
go install github.com/gregory-chatelier/watchman@latest
```

## Usage

The command structure is:

```bash
watchman [OPTIONS] -- [SUCCESS_COMMAND]
```

### Options

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-c`, `--command` | The command to execute and inspect. | |
| `-f`, `--file` | The path to the file to read and inspect. | |
| `-p`, `--pattern` | The exact string to search for in the output or file content. **Required.** | |
| `--interval` | The initial interval between polling attempts (e.g., `5s`, `1m`). | `1s` |
| `--max-retries` | Maximum polling attempts before giving up. `0` means retry forever. | `10` |
| `--backoff` | Exponential backoff factor (delay is multiplied by this factor each retry). A factor of `1` disables exponential backoff. | `1` |
| `--timeout` | Overall max wait time (e.g., `5m`). Overrides `--max-retries`. | `0` (no timeout) |
| `--on-fail` | Command to execute if the pattern is not found after all attempts or on timeout. | |
| `-v`, `--verbose` | Enable verbose logging. | `false` |

## Examples

### 1. Wait for a Service Health Check

Polls a health endpoint every 5 seconds, with a backoff factor of 2, up to 10 times. If successful, it runs the test suite. If it fails, it sends an alert.

```bash
watchman \
  -c "curl -s https://api.myservice.com/health" \
  -p '"status":"green"' \
  --max-retries 10 \
  --interval 5s \
  --backoff 2 \
  --on-fail "echo 'Service never became healthy' | mail -s 'Deploy failed' ops@company.com" \
  -- ./run_tests.sh
```

### 2. Wait for a Log Message

Monitors a build log for the success message, timing out after 5 minutes.

```bash
watchman \
  --file "./build.log" \
  --pattern "BUILD SUCCESSFUL" \
  --timeout 5m \
  --on-fail "echo 'Build failed or timed out'" \
  -- ./deploy.sh
```
