.PHONY: help run dev-run fmt test vet lint check cover integration ci-debug dev-deps-up dev-deps-down dev-deps-reset dev-deps-ps dev-deps-logs stack-up stack-down stack-reset stack-ps stack-logs

help:
	@echo "Available commands:"
	@echo "  make run            - run app locally against default local deps"
	@echo "  make dev-deps-up    - start MySQL and Redis for host-side development"
	@echo "  make dev-run        - run app locally against Docker dev deps"
	@echo "  make dev-deps-down  - stop Docker dev deps"
	@echo "  make dev-deps-reset - stop Docker dev deps and remove volumes"
	@echo "  make check          - run fmt, tests, vet, and lint"
	@echo "  make integration    - run tagged real-dependency integration tests"
	@echo "  make ci-debug       - summarize the latest failed GitHub Actions log"
	@echo "  make stack-up       - start full Docker stack"
	@echo "  make stack-down     - stop full Docker stack"
	@echo "  make stack-reset    - stop full Docker stack and remove volumes"

run:
	sh scripts/run-local.sh

dev-run:
	sh scripts/run-dev.sh

dev-deps-up:
	sh scripts/dev-deps.sh up

dev-deps-down:
	sh scripts/dev-deps.sh down

dev-deps-reset:
	sh scripts/dev-deps.sh down-v

dev-deps-ps:
	sh scripts/dev-deps.sh ps

dev-deps-logs:
	sh scripts/dev-deps.sh logs

fmt:
	go fmt ./...

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: fmt test vet lint

cover:
	go test ./... -cover

integration:
	sh scripts/integration-check.sh

ci-debug:
	sh scripts/ci-debug.sh $(RUN_ID)

stack-up:
	docker compose up --build -d

stack-down:
	docker compose down

stack-reset:
	docker compose down -v

stack-ps:
	docker compose ps

stack-logs:
	docker compose logs -f app mysql redis
