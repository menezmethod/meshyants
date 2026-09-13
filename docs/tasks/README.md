# MeshyAnts Agent Task System

This folder is the operating surface for coding/research agents.

## The rule

**Do not build features because they sound interesting. Build experiments that can change our mind.**

`tasks.json` is the shared queue. Work in dependency order, but do not mechanically implement the backlog if new evidence invalidates it.

## Required loop

1. Pick the highest-priority unblocked task.
2. Read `docs/revival/*` and state the claim being tested.
3. Produce the smallest implementation/experiment that could falsify the claim.
4. Run the evaluator gauntlet in `EVALUATORS.md`.
5. Record evidence, update the task, then choose the next step.

A task can end as **rejected**. That is a valid result.

## Workstreams

- **Science:** baselines, experiments, falsification.
- **Systems:** performance, scalability, failure recovery.
- **NeuroAI:** translate current research into testable mechanisms.
- **Quality:** skeptical critics and anti-slop gates.
- **Community:** contribution model, portable ants, reproducibility.
- **North star:** the WOW demo that proves useful emergent/adaptive behavior.

## Important constraint

If a simpler scheduler, queue, DAG, or manager-worker system explains the result equally well, **MeshyAnts has not earned the complexity yet**.
