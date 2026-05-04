# Observability, Logging, And Error Reporting

This document defines the current v1.0 observability baseline. It does not add an external monitoring service or hosted error-reporting integration.

## Current Baseline

- Startup, MySQL database creation, Redis check, migration, and server listen failures are logged through the Go standard logger in `cmd/server`.
- HTTP request logs are emitted by Gin.
- User-facing errors are rendered in the web UI or returned as JSON for async endpoints.
- AI reply failures in async chat are saved as system messages in the chat history.
- Tool executions are persisted in `tool_executions`, including success or failure status.

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

## Future Enhancements

- Structured JSON logs with request IDs.
- Correlation IDs across async AI reply workers.
- Metrics for model latency, retry counts, token usage, and tool execution outcomes.
- External error tracking integration.
- Alerting for repeated model API failures, MySQL/Redis failure, and disk exhaustion.
