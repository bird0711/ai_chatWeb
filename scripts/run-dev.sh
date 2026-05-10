#!/usr/bin/env sh
set -eu

export MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
export MYSQL_PORT="${MYSQL_PORT:-${DEV_MYSQL_PORT:-3307}}"
export MYSQL_USER="${MYSQL_USER:-root}"
export MYSQL_PASSWORD="${MYSQL_PASSWORD:-4399}"
export MYSQL_DATABASE="${MYSQL_DATABASE:-ai_chat}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:${DEV_REDIS_PORT:-6380}}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-4399}"
export REDIS_DB="${REDIS_DB:-0}"

exec sh scripts/run-local.sh
