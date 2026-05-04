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

## Run

```sh
go mod tidy
sh scripts/run-local.sh
```

Open `http://localhost:8080`.

If port `8080` is already in use, `scripts/run-local.sh` automatically selects the first free port from `8081` to `8090` and prints the actual URL. You can also set a port manually:

```sh
ADDR=':9000' sh scripts/run-local.sh
```

On startup the app will create the `ai_chat` database if it does not exist, run table migrations, and check Redis connectivity.

## Checks

```sh
sh scripts/ci-check.sh
```

The same test/build gate is documented in `docs/ai/ci.md`.
