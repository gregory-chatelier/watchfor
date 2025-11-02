# watchfor

A smarter `watch` utility that can trigger actions not just on file change, but on specific content changes.

## 1. Summary

`watchfor` is a command-line utility designed to repeatedly run a command or read a file until specific content is detected -state-based-. It serves as an intelligent and resilient command orchestrator that aims to prevent race conditions in CI/CD pipelines.

It repeatedly polls for a *state* (e.g., a health check returning "healthy", a log file containing "BUILD SUCCESSFUL") and then triggers a subsequent action and allows to handle transient failures and create robust automation.

It features:
*   **Command Polling:** Repeatedly runs a command (e.g., a health check) and inspects its output.
*   **File Watching:** Efficiently monitors a file (like a log) for new content, similar to `tail -f`.
*   **Resilience:** Built-in support for maximum retries, exponential backoff, and a global timeout.
*   **Graceful Failure:** Executes a fallback command (`--on-fail`) if the condition is never met.

## 2. Core Concepts

`watchfor` operates in one of two modes:

1.  **Command Mode (`-c` or `--command`):** Executes a shell command at a regular interval and inspects its standard output. This is the primary mode for polling health checks or API endpoints.
2.  **File Mode (`-f` or `--file`):** Reads the content of a specified file at a regular interval. This is useful for monitoring log files or build artifacts.

In both modes, if the pattern specified by `-p` is found, `watchfor` executes a success command. If the pattern is not found after all retries, it executes a failure command.

## Usage

The command structure is:

```bash
watchfor [OPTIONS] -- [SUCCESS_COMMAND]
```

### Options

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-c`, `--command` | The command to execute and inspect. | |
| `-f`, `--file` | The path to the file to read and inspect. | |
| `-p`, `--pattern` | The exact string to search for in the output or file content. **Required.** | |
| `--regex` | Enable regex matching for the pattern. | `false` |
| `--ignore-case` | Enable case-insensitive matching for the pattern. | `false` |
| `--interval` | The initial interval between polling attempts (e.g., `5s`, `1m`). | `1s` |
| `--max-retries` | Maximum polling attempts before giving up. `0` means retry forever. | `10` |
| `--backoff` | Exponential backoff factor (delay is multiplied by this factor each retry). A factor of `1` disables exponential backoff. | `1` |
| `--timeout` | Overall max wait time (e.g., `5m`). Overrides `--max-retries`. | `0` (no timeout) |
| `--on-fail` | Command to execute if the pattern is not found after all attempts or on timeout. | |
| `-v`, `--verbose` | Enable verbose logging. | `false` |

### Pattern Matching Details

When using the `--regex` flag, `watchfor` utilizes Go's standard regular expression syntax. You can find detailed documentation on the supported regex syntax [here](https://pkg.go.dev/regexp).


## Installation

This single command will download and install `watchfor` to a sensible default location for your system.

**User-level Installation (Recommended for most users):**
Installs `watchfor` to `$HOME/.local/bin` (Linux/macOS) or a user-specific `bin` directory (Windows).

```bash
curl -sSfL https://raw.githubusercontent.com/gregory-chatelier/watchfor/main/install.sh | sh
```

**System-wide Installation (Requires `sudo`):**
Installs `watchfor` to `/usr/local/bin` (Linux/macOS).

```bash
sudo curl -sSfL https://raw.githubusercontent.com/gregory-chatelier/watchfor/main/install.sh | sh
```

**Custom Installation Directory:**

You can specify a custom installation directory using the `INSTALL_DIR` environment variable:

```bash
curl -sSfL https://raw.githubusercontent.com/gregory-chatelier/watchfor/main/install.sh | INSTALL_DIR=$HOME/bin sh
```

## Examples

### 1. Wait for a Service Health Check

Polls a health endpoint every 5 seconds, with a back-off factor of 2, up to 10 times. If successful, it runs the test suite. If it fails, it sends an alert.

```bash
watchfor \
  -c "curl -s https://api.myservice.com/health" \
  -p '"status":"green"' \
  --max-retries 10 \
  --interval 5s \
  --backoff 2 \
  --on-fail "echo 'Service never became healthy' | mail -s 'Deploy failed' ops@company.com; exit 1" \
  -- ./run_tests.sh
```

### 2. Wait for a Log Message

Monitors a build log for the success message, timing out after 5 minutes.

```bash
watchfor \
  --file "./build.log" \
  --pattern "BUILD SUCCESSFUL" \
  --timeout 5m \
  --on-fail "echo 'Build failed or timed out'; exit 1" \
  -- ./deploy.sh
```

### 3. Use case : prevent CI/CD Race Conditions

`watchfor` is designed to replace time-based waiting loops with resilient, state-based polling.

Before: The common pattern, a shell loop with sleep instructions

```bash
script:
#...
  - echo "Health check..."
  - |
    for i in $(seq 1 10); do
      if curl -f http://$HOST:$PORT/health; then
        echo "Service is active!"
        break
      else
        echo "Attempt $i/10: Service not yet active. Waiting 5 seconds..."
        sleep 5
      fi
    done
    if ! curl -f http://$HOST:$PORT/health; then
      echo "Health check failed after multiple attempts."
      exit 1
    fi
```

After: One single declarative and resilient command

```bash
script:
#...
  - echo "Health check: waiting for service to become active..."
  - |
    watchfor \
      -c "curl -sf http://$HOST:$PORT/health" \
      -p "200" \
      --max-retries 10 \
      --interval 5s \
      --on-fail "echo '❌ Health check failed after multiple attempts.'; \
                 exit 1" \
      -- echo "✅ Service is active!"
```

### 4. Wait for a Kubernetes Pod to be Ready (Regex and Case-Insensitive)

Polls `kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].ready}'` until it returns `true`, ignoring case.

```bash
watchfor \
  -c "kubectl get pod my-app-pod -o jsonpath='{.status.containerStatuses[0].ready}'" \
  -p "true" \
  --ignore-case \
  --timeout 2m \
  --on-fail "echo 'Pod did not become ready in time'; exit 1" \
  -- echo "Kubernetes pod is ready!"
```

### 5. Monitor Docker Container Logs for a Specific Error (Regex)

Monitors a Docker container's logs for a specific error pattern, using regex to match variations.

```bash
watchfor \
  -c "docker logs my-container" \
  -p "(ERROR|FAILURE): .* (failed to connect|connection refused)" \
  --regex \
  --interval 5s \
  --max-retries 60 \
  --on-fail "echo 'Specific error pattern not found in logs'; exit 1" \
  -- echo "Error pattern detected in Docker logs."
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
