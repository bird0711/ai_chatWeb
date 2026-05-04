#!/usr/bin/env sh
set -eu

: "${GOCACHE:=/tmp/go-build-ai-chat}"
export GOCACHE

go test -mod=mod ./...
go build -mod=mod -buildvcs=false ./cmd/server
