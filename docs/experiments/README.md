# Experiment contract

Normative comparison rules for v2 live in [`CONTRACT.md`](CONTRACT.md).

This folder is the Phase 0 lock for MESH-101. It does **not** implement the harness (MESH-103), worker phenotypes/leases (MESH-102), or MeshyAnts allocation (MESH-104).

| Path | Role |
|---|---|
| `CONTRACT.md` | Normative rules: held-constant set, metrics, budget, comparison, falsification |
| `schema/` | JSON Schema for run specs, task sets, worker pools, and run results |
| `examples/phase-a-synthetic/` | Labeled calibration DAG (not decisive) |
| `examples/phase-a-uncertain/` | Decisive Phase A fixture: empty or wrong visible labels |

Validate a run spec:

```bash
go run ./cmd/expcontract validate docs/experiments/examples/phase-a-synthetic/run.json
```
