# AI Chat Groups

v0.1 MVP is a Gin-based Go web app for a single-user AI group chat flow.

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
#
# MYSQL_DSN is optional. If it is not set, the app creates ai_chat automatically.
export REDIS_ADDR='127.0.0.1:6379'
export REDIS_PASSWORD='4399'
export REDIS_DB='0'
export ADDR=':8080'
```

The model API settings are entered in the web UI at `/settings/model`.

For the full local configuration reference, see `docs/ai/developer-settings.md`.

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
