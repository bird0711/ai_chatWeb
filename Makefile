.PHONY: run fmt test vet lint check cover

run:
	sh scripts/run-local.sh

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

