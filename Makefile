.PHONY: build test run lint fmt

build:
	CGO_ENABLED=1 go build -o synapse ./cmd/synapse

test:
	CGO_ENABLED=1 go test ./...

run:
	CGO_ENABLED=1 go run ./cmd/synapse
