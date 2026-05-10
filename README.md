# AI Chat Groups

AI Chat Groups is a Gin-based Go web app for account-isolated AI group chats with configurable AI roles, model API settings, token usage statistics, and early v1.0 file-assisted analysis.

## Required Services

- MySQL
- Redis
- OpenAI-compatible chat completions endpoint

## Environment

```sh
# Defaults match the current local MVP setup:
# MYSQL_USER=root
# MYSQL_PASSWORD=4399
# MYSQL_HOST=127.0.0.1
# MYSQL_PORT=3306
# MYSQL_DATABASE=ai_chat
# REDIS_ADDR=127.0.0.1:6379
# REDIS_PASSWORD=4399
# REDIS_DB=0
# CHAT_FILE_DIR=data/chat-files
# MODEL_API_TIMEOUT_SECONDS=90
# MODEL_API_TLS_HANDSHAKE_TIMEOUT_SECONDS=30
# MODEL_API_RETRY_ATTEMPTS=2
# MODEL_API_RETRY_BACKOFF_MS=800
#
# MYSQL_DSN is optional. If it is not set, the app creates ai_chat automatically.
export REDIS_ADDR='127.0.0.1:6379'
export REDIS_PASSWORD='4399'
export REDIS_DB='0'
export ADDR=':8080'
```

The model API settings are entered in the web UI at `/settings/model`.

Supported chat analysis files can be uploaded directly from the local computer on a chat page. The current v1.0 slice supports `txt`, `md`, `json`, `csv`, `log`, `docx`, and text-based `pdf` files up to 10MB, stored by default under `data/chat-files` and injected into AI reply context. Scanned image-only PDFs are not supported.

For the full local configuration reference, see `docs/ai/developer-settings.md`.

For self-hosted deployment guidance, see `docs/ai/deployment.md`.

For contributor workflow guidance, see `CONTRIBUTING.md`.

For manual release and handoff verification, see `docs/ai/release-checklist.md`.

## Workflows

Use the project through one of these four entrypoints:

### 1. Local development

Recommended for daily coding.

Start only MySQL and Redis in Docker:

```sh
make dev-deps-up
```

Run the Go app locally against those Docker dependencies:

```sh
make dev-run
```

Equivalent explicit form:

```sh
MYSQL_PORT=3307 REDIS_ADDR=127.0.0.1:6380 make run
```

Useful companion commands:

```sh
make dev-deps-ps
make dev-deps-logs
make dev-deps-down
make dev-deps-reset
```

What this mode is for:

- fastest daily development loop
- Go code runs on the host for easier editing, debugging, and Git work
- MySQL and Redis stay reproducible inside Docker
- development dependencies use host ports `3307` and `6380` by default to avoid collisions with an existing local MySQL or Redis

Quick command summary:

```sh
make help
```

### 2. Daily checks

```sh
make check
```

This runs formatting, unit/default tests, vet, and golangci-lint.

### 3. Integration tests

```sh
make integration
```

This starts test-only MySQL and Redis containers, runs tagged real-dependency tests, and cleans everything up automatically.

### 4. Full containerized stack

```sh
make stack-up
```

Useful companion commands:

```sh
make stack-ps
make stack-logs
make stack-down
make stack-reset
```

What this mode is for:

- validating the full Docker delivery path
- checking that `app`, `mysql`, and `redis` run together correctly
- reproducing the project on a clean machine
- handoff to other contributors who do not want to install MySQL and Redis manually

Before treating a branch as handoff-ready, run the checklist in `docs/ai/release-checklist.md`.

## Run

```sh
make run
```

Open `http://localhost:8080`.

If port `8080` is already in use, `scripts/run-local.sh` automatically selects the first free port from `8081` to `8090` and prints the actual URL. You can also set a port manually:

```sh
ADDR=':9000' sh scripts/run-local.sh
```

On startup the app will create the `ai_chat` database if it does not exist, run table migrations, and check Redis connectivity.

## Quality Gate

Before commit, run:

```sh
make check
```

Equivalent commands:

```sh
go fmt ./...
go test ./...
go vet ./...
golangci-lint run ./...
```

Current local status:

- `go test ./...` passes
- `go vet ./...` passes
- `golangci-lint run ./...` passes

Latest coverage snapshot (`go test ./... -cover`):

- `internal/store`: `73.0%`
- `internal/app`: `70.4%`
- `internal/ai`: `59.9%`
- `internal/http`: `58.0%`

The same test/build gate is documented in `docs/ai/ci.md`.
Contributor workflow details are documented in `CONTRIBUTING.md`.
Release and handoff verification is documented in `docs/ai/release-checklist.md`.

## Integration Tests

The repository includes tagged integration tests for real MySQL and Redis dependencies.

Recommended command:

```sh
make integration
```

Equivalent command:

```sh
sh scripts/integration-check.sh
```

What it does:

- Starts `mysql` and `redis` in test mode with `docker-compose.test.yml`
- Exposes host port `3307` for MySQL and `6380` for Redis by default to avoid common local conflicts
- Runs `go test -tags=integration -mod=mod ./internal/store ./internal/http`
- Stops and removes the test containers and volumes when finished

Coverage included in the integration test flow:

- real MySQL store operations
- real Redis connectivity
- HTTP main flow over real dependencies: register/login, create chat, add role, send message, upload file, view usage, health check
