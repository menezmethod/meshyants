# Research Thesis

## Updated scorecard

| Ingredient | Current MeshyAnts interpretation | Confidence | Main gap |
|---|---|---:|---|
| **U — Units** | Reproducible worker phenotypes; LLM or deterministic tool | High | Best unit granularity |
| **T — Topology** | Local modules + sparse bridges + recurrence + integrators/broadcasters | Medium-high | Which topology wins for which workload |
| **F — Feedback** | Objective verification + positive/negative signals + cost | High | Reward hacking / evaluator quality |
| **M — Memory** | Working, episodic, semantic, procedural, structural, evolutionary | Medium-high | Retention/decay policy |
| **L — Learning** | Adaptive thresholds, contextual reputation, bandits, value-of-learning | Medium | Credit assignment / concept drift |
| **E — Environment** | Real tools, artifacts, APIs, users, downstream outcomes | High | Safe effect boundaries |
| **C — Competition/cooperation** | Stochastic claims + inhibition + shared artifacts | Medium | Exploration pressure |
| **S — Selection pressure** | More resources/replication for configurations that produce verified value | Medium | Fitness design is dangerous |
| **t — Time** | Fast task dynamics; slower reputation/topology/population adaptation | Medium-high | Correct timescales |

## Evidence that changed the design

1. **FlyWire whole-brain topology:** rich-club organization, integrator/broadcaster-like nodes, recurrence and short paths argue against a flat global swarm.
2. **2026 brain+cord connectome:** strongest control is local feedback; long-range circuits coordinate behavior-centric modules; higher regions supervise rather than micromanage.
3. **Princeton Cognitive Legos:** reusable cognitive components support compositional skills rather than endless handcrafted “roles.”
4. **Princeton effort allocation:** agents should sometimes spend effort where expected information gain is high, not only where immediate success probability is highest.
5. **Evolutionary MAS research:** selection + mutation/crossover in structured agent-configuration space makes **S** worth testing, but not trusting blindly.

## What must be true for MeshyAnts to matter?

- Coordination gains must exceed coordination overhead.
- Benefits must survive comparison to a competent centralized router.
- Learning must improve future allocation without locking into early winners.
- Selection pressure must correlate with real outcomes, not proxy gaming.
- Topology must provide value beyond a dressed-up priority queue.

## Evidence that would change the conclusion

MeshyAnts should be considered unsuccessful if, under equal budgets, a simple manager/router consistently wins on quality, cost, latency, resilience and adaptation across the target workload classes.

## Current strongest hypothesis

MeshyAnts is **unlikely to win universally**. The plausible win condition is dynamic, heterogeneous, failure-prone work where expertise is uncertain and new information creates new work.
