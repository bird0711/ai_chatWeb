# CI Checks

This document defines the current v1.0 CI quality gate.

## Required Checks

The minimum CI gate runs:

```sh
go test -mod=mod ./...
go build -mod=mod -buildvcs=false ./cmd/server
```

These checks do not require MySQL, Redis, a model API key, browser automation, or deployment credentials.

## Local Command

From the repository root:

```sh
sh scripts/ci-check.sh
```

The script sets `GOCACHE` to `/tmp/go-build-ai-chat` by default and runs the same checks as CI.

## GitHub Actions

The workflow is stored at:

```text
.github/workflows/ci.yml
```

It uses `go-version-file: go.mod`, then runs the test and build commands. If the hosted runner does not yet provide the exact Go version from `go.mod`, run `scripts/ci-check.sh` locally until the runner image supports it.

## Out Of Scope

- Browser end-to-end tests.
- MySQL/Redis integration tests.
- Real model API tests.
- Deployment.
- Secret-dependent checks.
