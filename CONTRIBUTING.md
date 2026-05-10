# Contributing

This repository currently uses three different working modes. Pick the mode that matches the job you are doing instead of defaulting to full Docker for everything.

## Recommended Daily Workflow

1. Start development dependencies:

```sh
make dev-deps-up
```

This starts MySQL and Redis in Docker on host ports `3307` and `6380`. The purpose is to keep the app dependencies reproducible without forcing the Go app itself into a container during daily coding.

2. Run the Go app on the host:

```sh
make dev-run
```

This keeps editing, debugging, file watching, and Git operations simple while still using the same database and cache shape every contributor can reproduce.

3. Run the default quality gate before commit:

```sh
make check
```

This runs formatting, default tests, `go vet`, and `golangci-lint`.

4. Run real-dependency integration tests when you change startup, HTTP flows, MySQL code, Redis code, or test infrastructure:

```sh
make integration
```

5. Validate the full delivery path when you touch Docker or release-facing startup behavior:

```sh
make stack-up
make stack-logs
make stack-down
```

## Working Modes

### Host app + Docker dependencies

Use this for most development.

- Start dependencies: `make dev-deps-up`
- Run app: `make dev-run`
- Stop dependencies: `make dev-deps-down`

### Full Docker stack

Use this to verify that a clean machine can run the project end to end.

- Start full stack: `make stack-up`
- Check status: `make stack-ps`
- View logs: `make stack-logs`
- Stop stack: `make stack-down`
- Remove stack volumes: `make stack-reset`

### Pure local services

Use this only if you intentionally want to run MySQL and Redis outside Docker.

```sh
make run
```

If your local MySQL and Redis are on non-default addresses, export the matching environment variables first.

## What To Run For Different Changes

- UI/template change: `make check`
- Store or Redis change: `make check` and `make integration`
- HTTP handler or auth flow change: `make check` and `make integration`
- Dockerfile or Compose change: `make integration` and `make stack-up`
- Startup/config change: `make check`, `make integration`, and usually `make stack-up`

## Notes

- Integration tests are intentionally tagged with `integration`, so `go test ./...` stays fast.
- The project keeps MySQL and Redis on separate Docker Compose override files for development and test use cases.
- Uploaded files and runtime data are ignored by Git and should not be committed.
