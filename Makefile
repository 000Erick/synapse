.PHONY: build test run lint fmt

build:
	CGO_ENABLED=0 go build -o synapse ./cmd/synapse

test:
	CGO_ENABLED=0 go test ./...

run:
	CGO_ENABLED=0 go run ./cmd/synapse
