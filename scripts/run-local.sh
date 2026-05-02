#!/usr/bin/env sh
set -eu

export MYSQL_USER="${MYSQL_USER:-root}"
export MYSQL_PASSWORD="${MYSQL_PASSWORD:-4399}"
export MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
export MYSQL_PORT="${MYSQL_PORT:-3306}"
export MYSQL_DATABASE="${MYSQL_DATABASE:-ai_chat}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-4399}"
export REDIS_DB="${REDIS_DB:-0}"

if [ "${ADDR:-}" = "" ]; then
  for port in 8080 8081 8082 8083 8084 8085 8086 8087 8088 8089 8090; do
    if ! ss -tuln | grep -q ":${port} "; then
      export ADDR=":${port}"
      break
    fi
  done
fi

if [ "${ADDR:-}" = "" ]; then
  echo "No free port found in 8080-8090. Set ADDR manually, for example: ADDR=:9000 sh scripts/run-local.sh" >&2
  exit 1
fi

echo "Starting AI Chat Groups at http://localhost${ADDR}"

exec go run -mod=mod -buildvcs=false ./cmd/server
