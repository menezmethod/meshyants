# Experiments

## Phase A — isolate routing

Hold models/tools/tasks/budget constant.

Compare:

1. **Single best worker**
2. **Manager → workers**
3. **Central capability router**
4. **Flat blackboard volunteers**
5. **MeshyAnts:** local topology + thresholds + contextual reputation + inhibition + exploration

Measure: task success, quality, wall time, token/$ cost, duplicate work, recovery time and coordination overhead.

## Phase B — topology

Same workers, different wiring:

- flat
- manager tree
- modular local neighborhoods
- modular + sparse integrators/broadcasters
- recurrent modular network

Question: **does topology create measurable capability beyond individual workers?**

Define an emergence delta:

`E = colony_quality - best_single_worker_quality`

## Phase C — adaptation

Introduce:
- new cold-start worker
- degraded formerly-good worker
- workload distribution shift
- node failure
- demand burst

Test whether reputation decay, exploration and local inhibition adapt without manual routing.

## Phase D — selection pressure

Only after A–C work.

Compare:
- no selection
- contextual routing only
- resource/replica selection
- controlled phenotype mutation + selection

Fitness must include quality, verification confidence, cost, latency and reliability. Track failure modes caused by Goodharting.

## First real workload

Use **SonicTale** for durable workflow/failure experiments, but do not treat it as the decisive swarm benchmark: its write → TTS → finalize path is naturally DAG-like and may favor deterministic orchestration.
