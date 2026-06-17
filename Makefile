.PHONY: build test run lint fmt

build:
	CGO_ENABLED=1 go build -tags sqlite_fts5 -o synapse ./cmd/synapse

test:
	CGO_ENABLED=1 go test -tags sqlite_fts5 ./...

run:
	CGO_ENABLED=1 go run -tags sqlite_fts5 ./cmd/synapse
