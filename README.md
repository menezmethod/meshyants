# MeshyAnts

**Experimental substrate for collective machine intelligence.**

MeshyAnts asks one falsifiable question:

> Given the same models, tools, tasks, and budget, can decentralized biologically inspired coordination outperform conventional agent orchestration on quality, cost, resilience, or adaptation?

The existing Go v1 already contains useful substrate: signed `TaskAtom`s, `CapabilityAdvertisement`s, `PheromoneRecord`s with decay, `ReputationEvent`s, trust domains, a ledger, Oracle/Queen components, routing, failure handling, and NATS-oriented runtime pieces.

## Research model

`CI = f(U, T, F, M, L, E, C, S, t)`

- **U — Units:** agents/tools + capabilities
- **T — Topology:** who can influence whom
- **F — Feedback:** verification, reward, inhibition
- **M — Memory:** working → evolutionary
- **L — Learning:** thresholds, reputation, bandits, adaptation
- **E — Environment:** tools, code, APIs, users, real outcomes
- **C — Competition/cooperation:** claim work vs share results
- **S — Selection pressure:** what survives, gets resources, replicates, or disappears
- **t — Time:** decay, leases, drift, slow structural adaptation

Two cross-cutting constraints: **homeostasis** (prevent runaway activation/cost) and **integration** (turn local work into useful global behavior).

## Current status

This is a research project, not a claim that brains and AI agents are equivalent. v2 should first test coordination mechanisms against strong simpler baselines.

Start with:
- `docs/revival/RESEARCH.md`
- `docs/revival/ARCHITECTURE.md`
- `docs/revival/EXPERIMENTS.md`
- `docs/revival/ROADMAP.md`
- `docs/revival/MENTAT.md`
