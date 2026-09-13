# Pareto Roadmap

## Phase 0 — planning gate (now)

Do not add major v2 code until these are decided:

1. Exact hypothesis and baseline harnesses.
2. Worker phenotype schema.
3. Task/lease/checkpoint state machine.
4. Topology + signal semantics.
5. Evaluation metrics and falsification criteria.

Exit condition: another engineer can implement Experiment A without inventing missing architecture.

## Phase 1 — minimum experiment

Reuse as much v1 substrate as possible:
- `TaskAtom`
- capability ads
- pheromone records + decay
- ledger
- Oracle boundary
- failure/DLQ pieces

Add only what Experiment A needs: leases/fencing if incomplete, contextual capability labels, reproducible routing decisions, cost/latency evidence and benchmark harness.

## Phase 2 — topology

Add local neighborhoods, inhibition/saturation, integrator/broadcaster primitives and recurrent verifier loops.

## Phase 3 — learning

Add contextual reputation, decay, Thompson/bandit exploration and drift detection.

## Phase 4 — selection / Queen

Only now test reproduction/retirement/mutation of phenotypes. Queen controls population/configuration, never individual task assignment.

## Explicitly deferred

Marketplace UI, Kubernetes, third-party ants, autonomous production deployment and open-ended self-modification.
