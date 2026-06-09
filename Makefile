.PHONY: generate lint test test-integration test-all

PROTO_SRC := meshyants/v1/contracts.proto

# Requires: protoc, protoc-gen-go on PATH (go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)
generate:
	protoc -I . --go_out=. --go_opt=module=github.com/meshyants/meshyants/v1 $(PROTO_SRC)

lint:
	golangci-lint run ./...

test:
	go test -short -race ./...

test-integration:
	go test -tags=integration -race ./...

test-all:
	go test -race ./...
