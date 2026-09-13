# WOW Target

The project needs a target that would make an AI researcher care.

## Weak demo

"Twenty agents talked and produced an app."

That proves orchestration works, not MeshyAnts.

## Strong demo

A fixed set of heterogeneous workers receives an ambiguous workload. Without a central planner assigning each step, the system:

1. forms/adapts a useful work path from local signals;
2. specializes or reroutes as evidence accumulates;
3. recovers when a strong worker disappears or the workload changes;
4. beats a credible simpler baseline on at least one meaningful metric;
5. reproduces the result across multiple runs.

Best version: **a workflow appears that we did not hard-code and an ablation shows the learned topology/feedback mechanism caused the gain.**

## Required proof

Show:
- control vs MeshyAnts
- identical task/model/tool budgets
- costs and latency
- failures/negative cases
- topology or allocation trace
- ablation: remove the alleged mechanism and show what changes

If we cannot show this, do not call the behavior emergent.
