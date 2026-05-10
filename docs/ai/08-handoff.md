# Handoff

## Current Project Position

This repository is not in a "final closeout" state. It is being improved step by step into a stronger engineering baseline.

## Completed Engineering Steps

1. unified configuration loading
2. `main.go` switched to the config object
3. startup initialization split out of `main.go`
4. standard `http.Server` and graceful shutdown
5. HTTP layer file split by responsibility
6. web security baseline
7. containerization and one-command stack startup
8. real-dependency integration testing
9. local development workflow unification
10. CI workflow cleanup
11. contributor-facing documentation and handoff cleanup
12. release and operations polish
13. request-level observability baseline

## Current Step

Step 13 is complete. The next candidate is Step 14.

## Recommended Working Pattern

- Daily coding: `make dev-deps-up` then `make dev-run`
- Default checks: `make check`
- Real MySQL/Redis verification: `make integration`
- Full stack validation: `make stack-up`

## Important Project Conventions

- Host-side app plus Docker MySQL/Redis is the preferred development loop.
- Full Docker stack is mainly for validation, handoff, and reproducibility.
- Integration tests use the `integration` build tag so default tests stay fast.
- Docker development and test ports intentionally avoid common local conflicts.

## Useful Documents

- `README.md`
- `CONTRIBUTING.md`
- `docs/ai/developer-settings.md`
- `docs/ai/ci.md`
- `docs/ai/deployment.md`
- `docs/ai/release-checklist.md`
- `docs/ai/observability.md`
- `docs/ai/STATUS.md`
- `docs/ai/NEXT.md`
