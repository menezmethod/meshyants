# MeshyAnts v1 (Go)

Reference implementation aligned with `docs/v1/`.

## Prerequisites

- Go 1.23+
- `protoc` and `protoc-gen-go` for regenerating protobufs (`make generate`)

## Commands

```bash
make generate   # regenerate gen/meshyantsv1 from meshyants/v1/contracts.proto
go test -short -race ./...
go test -tags=integration -race ./...   # Docker (NATS)
go test -tags=e2e -race ./test/e2e/... # Docker, full blackboard round-trip
```

Binaries:

- `go run ./cmd/meshyants/ version|doctor|serve`
- `go run ./cmd/queen/` — emits `QUEEN_*` keys and signed `JoinGrant` / `ProvisioningManifest` JSON (dev only).

## Audit test IDs

Failure-oriented audit cases from `docs/v1/10-failure-oriented-design-audit.md` are referenced in test names (e.g. `U4`, `I1`, `C8`) and `*_audit_test.go` files.
