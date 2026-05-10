# Release Checklist

This document is the lightweight release and handoff checklist for the current repository state. It is meant for manual verification before merge, handoff, or self-hosted deployment.

## When To Use It

Run this checklist when one of these is true:

- you changed startup, config, Docker, MySQL, Redis, HTTP flow, or persistence behavior
- you want to hand the project to another contributor
- you want to validate that the current branch is in a releasable state

## 1. Code Quality Gate

Run:

```sh
make check
```

Purpose:

- confirms formatting is clean
- runs default Go tests
- runs `go vet`
- runs `golangci-lint`

Expected result:

- command exits successfully with no lint or test failures

## 2. Real Dependency Verification

Run:

```sh
make integration
```

Purpose:

- verifies the app works against real MySQL and Redis, not only fakes or mocks
- covers store behavior and the main HTTP path

Expected result:

- integration command exits successfully
- test containers are cleaned up automatically

## 3. Full Docker Delivery Check

Run:

```sh
make stack-up
make stack-ps
make stack-logs
```

Purpose:

- verifies the clean-machine startup path
- confirms `app`, `mysql`, and `redis` can run together through Docker Compose

Expected result:

- all three services are up
- app logs show successful startup, migration, and dependency connection
- browser can open `http://localhost:8080`

After verification:

```sh
make stack-down
```

If you intentionally want to clear volumes too:

```sh
make stack-reset
```

## 4. Manual Smoke Test

Check these user-visible paths:

1. Open `/login`.
2. Register a new account or log in with a test account.
3. Open `/health` and confirm MySQL and Redis both report healthy.
4. Open `/settings/model` and confirm an existing model config can be viewed or a new one can be tested.
5. Create a chat.
6. Add at least two AI roles.
7. Send a message and confirm AI replies appear.
8. Upload a small supported file and confirm it is accepted.
9. Open usage/statistics and confirm the page loads.

Purpose:

- catches the class of failures that pass build/test but still break the basic UI flow

## 5. Persistence Check

Confirm these expectations:

- MySQL data survives app restarts
- Redis is reachable and using the intended database/password
- `UPLOAD_DIR` is writable
- `CHAT_FILE_DIR` is writable
- Docker volume or host-path persistence is configured for any environment you plan to hand off

Purpose:

- prevents the common "it runs, but state disappears" handoff failure

## 6. Handoff Check

Before calling the branch ready, confirm:

- `README.md` still matches the actual workflow commands
- `CONTRIBUTING.md` still matches the recommended daily workflow
- deployment notes still match the current startup shape
- no secret or local-only path was accidentally committed

## Minimum Ready Definition

Treat the branch as release-ready for the current project stage only when:

1. `make check` passes
2. `make integration` passes
3. full Docker stack starts successfully
4. manual smoke test passes
5. persistence expectations are clear and working
