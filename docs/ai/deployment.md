# Deployment Guide

This document describes a stable self-hosted deployment shape for AI Chat Groups. It is not a record that production deployment has already been executed.

For a manual pre-release and handoff check before deployment, use `docs/ai/release-checklist.md`.

## Target Shape

- One Linux host or VM running the Go server as a long-running process.
- MySQL reachable from the app host.
- Redis reachable from the app host.
- A reverse proxy such as Nginx or Caddy terminating TLS and forwarding HTTP to the app.
- Persistent local directories for role avatars and chat analysis files.
- Model API configs entered through the web UI after the first account logs in.

## Build

From the repository root:

```sh
go mod download
go build -mod=mod -buildvcs=false -o server ./cmd/server
```

Keep the built `server` binary, `web/templates`, and `web/static` together unless `TEMPLATE_GLOB` and `STATIC_DIR` point to different absolute paths.

## Required Services

MySQL:

- Create a dedicated database user for the app.
- Grant that user access to the configured database.
- The app can create the database when using individual `MYSQL_*` variables, but production deployments should normally provision the database explicitly.

Redis:

- Configure a password unless Redis is only reachable over a private network.
- Use a dedicated Redis database number when sharing Redis with other apps.

Model API:

- The web UI expects OpenAI-compatible endpoints.
- Model detection calls `GET {base_url}/models`.
- Chat replies call `POST {base_url}/chat/completions`.

## Environment

Use explicit production values rather than local defaults:

```sh
export ADDR='127.0.0.1:8080'
export MYSQL_DSN='ai_chat_user:strong-password@tcp(mysql-host:3306)/ai_chat?parseTime=true&multiStatements=true&charset=utf8mb4,utf8'
export REDIS_ADDR='redis-host:6379'
export REDIS_PASSWORD='strong-redis-password'
export REDIS_DB='0'
export TEMPLATE_GLOB='/opt/ai-chat/web/templates/*.html'
export STATIC_DIR='/opt/ai-chat/web/static'
export UPLOAD_DIR='/var/lib/ai-chat/uploads'
export CHAT_FILE_DIR='/var/lib/ai-chat/chat-files'
export MODEL_API_TIMEOUT_SECONDS='90'
export MODEL_API_TLS_HANDSHAKE_TIMEOUT_SECONDS='30'
export MODEL_API_RETRY_ATTEMPTS='2'
export MODEL_API_RETRY_BACKOFF_MS='800'
```

Do not commit production secrets. Store them in the process manager, host secret store, or deployment platform configuration.

## Persistent Data

Back up these stores:

- MySQL database: users, sessions, chats, roles, model configs, messages, token usage, chat file metadata, tool executions.
- `UPLOAD_DIR`: role avatar files served under `/uploads`.
- `CHAT_FILE_DIR`: private chat analysis files not served as static assets.

The `data/` and `uploads/` directories are intentionally ignored by Git.

## Process Manager

Run the app under a process manager such as systemd, supervisord, Docker, or a platform service runner. A minimal systemd unit can look like:

```ini
[Unit]
Description=AI Chat Groups
After=network.target mysql.service redis.service

[Service]
WorkingDirectory=/opt/ai-chat
ExecStart=/opt/ai-chat/server
Environment=ADDR=127.0.0.1:8080
Environment=MYSQL_DSN=ai_chat_user:strong-password@tcp(127.0.0.1:3306)/ai_chat?parseTime=true&multiStatements=true&charset=utf8mb4,utf8
Environment=REDIS_ADDR=127.0.0.1:6379
Environment=REDIS_PASSWORD=strong-redis-password
Environment=REDIS_DB=0
Environment=TEMPLATE_GLOB=/opt/ai-chat/web/templates/*.html
Environment=STATIC_DIR=/opt/ai-chat/web/static
Environment=UPLOAD_DIR=/var/lib/ai-chat/uploads
Environment=CHAT_FILE_DIR=/var/lib/ai-chat/chat-files
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Create persistent directories before starting:

```sh
mkdir -p /var/lib/ai-chat/uploads /var/lib/ai-chat/chat-files
```

## Reverse Proxy And TLS

Bind the app to loopback, for example `ADDR=127.0.0.1:8080`, and put TLS in front of it.

Example Nginx location:

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Set body size high enough for current uploads. Chat analysis files allow up to 10 MB and avatars allow up to 2 MB.

## Startup Verification

After starting the service:

1. Open `/login`.
2. Register or log in.
3. Open `/health` and confirm MySQL and Redis are healthy.
4. Open `/settings/model`, add a model API config, and run model detection.
5. Create a chat, add at least two AI roles, and send a message.
6. Upload a small text file and ask the AI to summarize it.
7. Run the calculator tool with `12 * (3 + 4)` and confirm result `84`.

This startup verification is the deployment-side subset of the broader release checklist in `docs/ai/release-checklist.md`.

## Backup And Restore

Recommended backup set:

- MySQL dump or physical backup.
- `UPLOAD_DIR`.
- `CHAT_FILE_DIR`.
- The deployed binary and exact source revision or release artifact.
- The deployment environment variable set, excluding secrets from general logs.

Restore order:

1. Restore MySQL.
2. Restore upload and chat-file directories to the configured paths.
3. Start Redis.
4. Start the app with the same environment.
5. Check `/health`, then verify a chat history page and one uploaded file entry.

## Rollback

Before upgrading:

- Build the new binary separately.
- Back up MySQL and persistent directories.
- Keep the previous binary available.

Rollback process:

1. Stop the app process.
2. Restore the previous binary.
3. Restore database and file backups if the failed version ran migrations or wrote incompatible data.
4. Start the app.
5. Check `/health` and the chat main path.

## Operational Notes

- Current logs are process stdout/stderr and Gin request logs.
- Production observability, structured logs, alerting, and external error reporting are handled in a later v1.0 slice.
- CI configuration already exists in `.github/workflows/ci.yml`, with local command guidance in `docs/ai/ci.md`.
- Use `docs/ai/observability.md` for incident checks and troubleshooting flow.
- Do not expose the app directly to the public internet without TLS and a reverse proxy.
