#!/usr/bin/env sh
set -eu

compose_files="-f docker-compose.yml -f docker-compose.dev.yml"
action="${1:-up}"

case "$action" in
  up)
    docker compose $compose_files up -d mysql redis

    until docker compose $compose_files exec -T mysql mysqladmin ping -h 127.0.0.1 -uroot -p4399 --silent >/dev/null 2>&1
    do
      sleep 1
    done

    until docker compose $compose_files exec -T redis redis-cli -a 4399 ping >/dev/null 2>&1
    do
      sleep 1
    done

    echo "Dev dependencies are ready."
    echo "MySQL: 127.0.0.1:${DEV_MYSQL_PORT:-3307}"
    echo "Redis: 127.0.0.1:${DEV_REDIS_PORT:-6380}"
    ;;
  down)
    docker compose $compose_files down
    ;;
  down-v)
    docker compose $compose_files down -v
    ;;
  ps)
    docker compose $compose_files ps
    ;;
  logs)
    docker compose $compose_files logs -f mysql redis
    ;;
  *)
    echo "Usage: sh scripts/dev-deps.sh {up|down|down-v|ps|logs}" >&2
    exit 1
    ;;
esac
