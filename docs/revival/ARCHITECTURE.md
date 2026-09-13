# v2 Architecture Direction

## Rule 1: AI does cognition; software does coordination

Do not spend LLM calls on claiming, leases, heartbeats, idempotency, status, retries or routing math.

## Core layers

1. **Oracle boundary** — human language ↔ structured intent. Not the task manager.
2. **Field** — soft signals: TODO, INSIGHT, DANGER, CLAIM/SATURATION, RESULT. Signals decay.
3. **Hard ledger** — authoritative tasks, attempts, leases, fencing tokens, checkpoints, artifacts and side effects.
4. **Worker modules** — local capability neighborhoods; workers do not subscribe to the whole colony.
5. **Slow adaptation** — reputation, topology reinforcement and eventually Queen selection operate on slower timescales.

## Worker phenotype

A worker is a reproducible configuration, not a magical persistent personality:

`model/tool + instructions/policy + skills + permissions + context strategy + budget`

If one of those changes materially, version the phenotype.

## Topology

Default hypothesis:

- local capability modules
- sparse long-range bridges
- recurrent verifier/repair loops
- integrator nodes combine evidence
- broadcaster nodes propagate important cross-module signals
- no global manager assigning every task

## Selection pressure

Selection is separate from feedback.

Feedback says **what happened**.  
Selection says **what persists**.

Selection can affect:
- probability of receiving similar work
- resource budget
- replica count
- phenotype survival
- pathway reinforcement

Selection must be multi-objective and confidence-aware. Never optimize one scalar proxy such as “tasks/hour.”

## Hard invariants

- exclusive side effects use atomic leases + fencing tokens
- idempotent external effects
- bounded retries
- checkpointed work
- stale workers cannot overwrite current owners
- reproducible routing decisions and random seeds
- deterministic tools are first-class workers
