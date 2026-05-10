# Observability, Logging, And Error Reporting

This document defines the current v1.0 observability baseline. It does not add an external monitoring service or hosted error-reporting integration.

## Current Baseline

- Startup, MySQL database creation, Redis check, migration, and server listen failures are logged through the Go standard logger in `cmd/server`.
- HTTP requests are logged with request ID, method, path, status, duration, and client IP.
- User-facing errors are rendered in the web UI or returned as JSON for async endpoints.
- AI reply failures in async chat are saved as system messages in the chat history.
- Tool executions are persisted in `tool_executions`, including success or failure status.
- Panic recovery logs include request ID and stack trace, and the error page/JSON response exposes the same request ID back to the caller.

## Request IDs

The app now assigns or accepts `X-Request-ID` for every request.

What this enables:

- one failing browser request can be matched to one log line sequence
- HTML error pages can show the request ID to the user
- JSON error responses can return the same request ID for API-style debugging
- recovered panics can be traced back to a specific request

Current behavior:

- if the client already sends a valid `X-Request-ID`, the app keeps it
- otherwise the app generates one
- the response always includes `X-Request-ID`

## First Response Workflow

When something breaks, check in this order:

1. Confirm which working mode you are using:
   - host app plus Docker dependencies
   - full Docker stack
   - pure local services
2. Check whether the app process is up and listening.
3. Check `/health`.
4. Check startup or runtime logs.
5. Reproduce one minimal failing request in the browser.
6. Decide whether the failure is in app startup, MySQL, Redis, model API, upload handling, or a UI flow.

This order matters because many visible failures are dependency or startup problems, not handler logic bugs.

## Useful Commands

For host-side development:

```sh
make dev-deps-ps
make dev-deps-logs
```

For full Docker validation:

```sh
make stack-ps
make stack-logs
```

For automated verification:

```sh
make check
make integration
```

These commands are the fastest way to distinguish:

- app code regression
- dependency startup failure
- Docker wiring problem
- test-only failure versus runtime failure

## Minimum Logging Policy

Use process stdout/stderr as the deployment baseline. A process manager or hosting platform should collect these logs.

Log these events:

- startup service checks and fatal startup failures.
- HTTP errors rendered through the generic error page.
- chat-page action errors rendered back into the chat page.
- async endpoint JSON errors.
- async AI reply failures that become system messages.
- file upload and tool execution failures when they return through chat-page error rendering.

Do not log:

- raw API keys.
- session tokens.
- uploaded file full contents.
- full chat message history.
- password hashes.

## Error Reporting Strategy

Current error reporting is local and operator-facing:

- Server logs show route, status, and high-level error text.
- Chat system messages tell the user when AI replies fail.
- Tool records show tool failure status and error text.
- `/health` exposes MySQL and Redis status for manual checks.

Future external error reporting can forward process logs to a log aggregator or error tracking service, but it must preserve the no-secret logging rule.

## Operational Checks

During deployment or incident response:

1. Check process logs for startup failures.
2. Open `/health` to verify MySQL and Redis.
3. Send a chat message and inspect system messages if AI replies fail.
4. Check model API settings and selected role routes.
5. Check tool execution records for controlled tool failures.
6. Confirm `UPLOAD_DIR` and `CHAT_FILE_DIR` are writable if uploads fail.

## Failure Buckets

Use these buckets before changing code:

### Startup failure

Common signs:

- app exits immediately
- `make stack-up` leaves `app` restarting
- `/health` never becomes reachable

Check:

- MySQL DSN or `MYSQL_*` values
- Redis address/password/database
- missing template or static directories
- missing writable upload/data directories

### Dependency failure

Common signs:

- app starts but `/health` reports degraded state
- integration tests fail before app behavior is exercised

Check:

- whether MySQL and Redis are actually running
- whether the selected ports match the current mode
- whether Docker volumes or previous state caused unexpected data conflicts

### Model API failure

Common signs:

- chat page loads but AI replies fail
- model detection fails in `/settings/model`

Check:

- base URL shape
- API key validity
- upstream provider reachability
- timeout and retry configuration

### Upload or file-storage failure

Common signs:

- file submit returns an error
- upload appears accepted but file-dependent behavior later fails

Check:

- file type support
- file size limit
- write permission on `UPLOAD_DIR` or `CHAT_FILE_DIR`
- whether Docker bind mounts or volumes map to the intended paths

## Future Enhancements

- Correlation IDs across async AI reply workers.
- Metrics for model latency, retry counts, token usage, and tool execution outcomes.
- External error tracking integration.
- Alerting for repeated model API failures, MySQL/Redis failure, and disk exhaustion.
