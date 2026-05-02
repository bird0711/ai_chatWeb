# Developer Settings

This document describes the local development configuration for AI Chat Groups.

## Required Services

- MySQL, reachable by the app process.
- Redis, reachable by the app process.
- An OpenAI-compatible model API, configured in the web UI after login.

## Environment Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `ADDR` | `:8080` through the app, auto-selected by `scripts/run-local.sh` | HTTP listen address. |
| `MYSQL_DSN` | empty | Optional full MySQL DSN. When set, it overrides individual `MYSQL_*` values. |
| `MYSQL_USER` | `root` | MySQL user when `MYSQL_DSN` is not set. |
| `MYSQL_PASSWORD` | `4399` | MySQL password when `MYSQL_DSN` is not set. |
| `MYSQL_HOST` | `127.0.0.1` | MySQL host when `MYSQL_DSN` is not set. |
| `MYSQL_PORT` | `3306` | MySQL port when `MYSQL_DSN` is not set. |
| `MYSQL_DATABASE` | `ai_chat` | MySQL database name. The app creates it if missing. |
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis address. |
| `REDIS_PASSWORD` | `4399` | Redis password. Use an empty value only if Redis has no password. |
| `REDIS_DB` | `0` | Redis database index. |
| `TEMPLATE_GLOB` | `web/templates/*.html` | HTML template glob. Mostly for tests or custom packaging. |
| `STATIC_DIR` | `web/static` | Static asset directory. |
| `UPLOAD_DIR` | `uploads` | Local upload directory for AI role avatars. |

Use `.env.example` as the reference for local configuration. The current run script does not automatically load `.env`; export variables in your shell or prefix the run command.

## Local Startup

From the repository root:

```sh
go mod tidy
sh scripts/run-local.sh
```

The run script:

- sets local defaults for MySQL and Redis.
- chooses the first free port from `8080` to `8090` when `ADDR` is not set.
- starts the Gin server with `go run -mod=mod -buildvcs=false ./cmd/server`.

If all default ports are busy, choose a port manually:

```sh
ADDR=':9000' sh scripts/run-local.sh
```

## Database Behavior

When `MYSQL_DSN` is not set, startup uses the individual `MYSQL_*` variables, creates the configured database if needed, opens MySQL, and runs migrations.

Current migrations create or update:

- users and sessions.
- chats, including AI review and topic fields.
- model API configs.
- roles, including model config binding, speaking permission, and avatar.
- messages.

## Redis Behavior

Startup checks Redis with the configured address, password, and database. Redis must be reachable before the app starts.

## Model API Settings

Model API settings are configured in the web UI at:

```text
/settings/model
```

The app expects an OpenAI-compatible API:

- model detection calls `GET {base_url}/models`.
- chat replies call `POST {base_url}/chat/completions`.
- `base_url` should normally look like `https://provider.example/v1`, not a full `/chat/completions` URL.
- the API key is sent as `Authorization: Bearer <api_key>`.

After a successful connection check, save the config and choose the saved API config/model when creating or editing AI roles.

## Uploads

AI role avatars are stored under:

```text
uploads/avatars/
```

Allowed image extensions:

- `.jpg`
- `.jpeg`
- `.png`
- `.gif`
- `.webp`

The maximum avatar file size is 2 MB. The `uploads/` directory is ignored by Git.

## Browser Settings

Theme mode is client-side only:

- the toggle is shown in the main navigation.
- the selected mode is stored in browser `localStorage` under `ai-chat-theme`.
- clearing browser site data resets the theme choice.

Enter-to-send is also client-side:

- plain Enter sends the chat message.
- Shift + Enter inserts a newline.
- IME composition is guarded to avoid accidental sends.

## Common Troubleshooting

### MySQL socket errors

If startup reports a MySQL TCP socket error, confirm MySQL is running and reachable from the same environment that starts the app:

```sh
mysql -h127.0.0.1 -P3306 -uroot -p4399 -D ai_chat
```

### Redis connection errors

Confirm Redis is running and the password/database match:

```sh
redis-cli -h 127.0.0.1 -p 6379 -a 4399 ping
```

### Port already in use

Use a different port:

```sh
ADDR=':9000' sh scripts/run-local.sh
```

### Model detection fails

Check:

- `base_url` ends at the API root, usually `/v1`.
- the provider supports `GET /models`.
- the API key is valid.
- the upstream provider account has access to the target models.

### Chat replies fail after model detection succeeds

Check:

- each AI role uses a saved API config/model option.
- at least two AI roles are allowed to speak.
- the selected model supports chat completions.
- the chat page system message for the exact model API error.
