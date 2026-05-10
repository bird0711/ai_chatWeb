# AI Coding Status

## Current Task Type

Incremental engineering improvement pass.

## Current Phase

Step 13 completed: request-level observability baseline.

## Current Goal

Keep the repository easier to release, hand off, and troubleshoot with explicit checklists plus request-level diagnostics.

## Completed Steps

1. Unified configuration loading.
2. `main.go` uses the unified config object.
3. Startup initialization split out of `main.go`.
4. Standard `http.Server` and graceful shutdown.
5. HTTP layer split from one large file into focused handlers/helpers.
6. Web security baseline: secure session cookie, CSRF, upload content validation.
7. Containerization: Dockerfile, Compose stack, dependency healthchecks, persistent volumes.
8. Real-dependency integration tests for MySQL, Redis, and the main HTTP flow.
9. Local development entrypoints unified around Make and Docker-backed dependencies.
10. CI structure cleaned up into `verify`, `build`, and `integration`.
11. Contributor-facing documentation and handoff cleanup.
12. Release checklist and operations/troubleshooting guidance.
13. Request ID and panic-recovery observability baseline.

## Current Working Model

- Preferred daily development: host-side Go app plus Docker MySQL/Redis.
- Full Docker stack: validation and handoff path, not the primary editing loop.
- Integration tests: real dependencies behind the `integration` build tag.

## Recent Verified Commands

- `make check`
- `make integration`
- `make dev-deps-up`
- `make dev-deps-ps`
- `make dev-deps-down`

## Current Risks

- Contributor-facing docs must stay aligned with the actual Make targets and scripts as future steps change workflows.
- Observability is better at request tracing now, but still lacks metrics, structured log output, and async correlation.

## Blockers

None.

## Next Recommendation

Start Step 14 by choosing between deeper observability follow-through or more automation around deployment/release workflow support.
