#!/usr/bin/env sh
set -eu

compose_files="-f docker-compose.yml -f docker-compose.test.yml"

cleanup() {
  docker compose $compose_files down -v
}

on_exit() {
  status=$?
  if [ "$status" -ne 0 ]; then
    echo "integration check failed with status $status" >&2
    docker compose $compose_files ps >&2 || true
    docker compose $compose_files logs mysql redis >&2 || true
  fi
  cleanup
  exit "$status"
}

trap on_exit EXIT INT TERM

docker compose $compose_files up -d mysql redis

until docker compose $compose_files exec -T mysql mysqladmin ping -h 127.0.0.1 -uroot -p4399 --silent >/dev/null 2>&1
do
  sleep 1
done

until docker compose $compose_files exec -T redis redis-cli -a 4399 ping >/dev/null 2>&1
do
  sleep 1
done

export MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
export MYSQL_PORT="${MYSQL_PORT:-${TEST_MYSQL_PORT:-3307}}"
export MYSQL_USER="${MYSQL_USER:-root}"
export MYSQL_PASSWORD="${MYSQL_PASSWORD:-4399}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:${TEST_REDIS_PORT:-6380}}"
export REDIS_PASSWORD="${REDIS_PASSWORD:-4399}"
export REDIS_DB="${REDIS_DB:-1}"

go test -tags=integration -mod=mod ./internal/store ./internal/http
