# CI Checks

This document defines the current repository CI quality gate.

## Required Checks

The default CI gate is split into two jobs:

### Verify

This job runs:

```sh
test -z "$(gofmt -l .)"
go test -mod=mod ./...
go vet ./...
golangci-lint run ./...
```

### Build

This job runs:

```sh
go build -mod=mod -buildvcs=false ./cmd/server
```

These jobs do not require MySQL, Redis, a model API key, browser automation, or deployment credentials.

## Integration Stage

CI also runs a separate integration stage:

```sh
sh scripts/integration-check.sh
```

This stage starts MySQL and Redis with Docker Compose and runs:

```sh
go test -tags=integration -mod=mod ./internal/store ./internal/http
```

The integration stage covers real dependency checks and the main HTTP flow on top of real MySQL and Redis.

## Local Commands

From the repository root:

```sh
make check
```

Run integration coverage separately when you change MySQL, Redis, HTTP main flows, or container/test wiring:

```sh
make integration
```

## GitHub Actions

The workflow is stored at:

```text
.github/workflows/ci.yml
```

It now uses three CI jobs:

- `verify`: format, default tests, vet, lint
- `build`: compile the server binary
- `integration`: start Docker test dependencies and run tagged real-dependency tests

If the hosted runner does not yet provide the exact Go version from `go.mod`, run `scripts/ci-check.sh` and `scripts/integration-check.sh` locally until the runner image supports it.

## Out Of Scope

- Browser end-to-end tests.
- Real model API tests.
- Deployment.
- Secret-dependent checks.
