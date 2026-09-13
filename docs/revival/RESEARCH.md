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

Primary citations and the mechanism → prediction map live in `docs/revival/NEUROAI_WATCH.md` (MESH-107). The bullets below are priors, not proofs. MESH-107 round 1 found no leftover coordination primitive beyond what a queue, router, DAG, manager, or subsumption stack already expresses; the watch’s residue is constraints and rejections, plus Experiment A vs those baselines.

1. **FlyWire whole-brain topology** (Lin et al., Nature 2024, [10.1038/s41586-024-07968-y](https://doi.org/10.1038/s41586-024-07968-y); dataset: Dorkenwald et al., [10.1038/s41586-024-07558-y](https://doi.org/10.1038/s41586-024-07558-y)): rich-club organization, a small *arbitrarily defined* integrator/broadcaster subset, over-represented reciprocity, and short paths. This argues against a *flat global sniff*, not for a rich-club controller — the fly rich club is ~30% of neurons, and later control-theory work treats rich clubs as highways more than commanders.
2. **2026 brain+cord connectome** (Bates, Phelps, Kim, Yang et al., Nature 2026, [10.1038/s41586-026-10735-w](https://doi.org/10.1038/s41586-026-10735-w)): strongest effector drive is local feedback; long-range AN/DN circuits coordinate behavior-centric modules; higher regions supervise rather than micromanage. The authors analogize to **subsumption / distributed robotic control**, which is a required simpler baseline, not a metaphor to copy wholesale.
3. **Princeton compositional subspaces** (Tafazoli et al., Nature 2025, [10.1038/s41586-025-09805-2](https://doi.org/10.1038/s41586-025-09805-2); press name “Cognitive Legos”): reusable PFC components support compositional *skills* rather than endless handcrafted roles. This is within-brain geometry, not a multi-agent org chart.
4. **Princeton effort allocation** (Masís Obando, Musslick, Cohen, PNAS 2025, [10.1073/pnas.2416720122](https://doi.org/10.1073/pnas.2416720122)): agents should sometimes spend effort where expected information gain is high, not only where immediate success probability is highest. Equal-budget MAS evidence (Kim et al., Nat. Mach. Intell. 2026, [10.1038/s42256-026-01268-y](https://doi.org/10.1038/s42256-026-01268-y)) says extra agents often *hurt* once a single capable worker exists.
5. **Evolutionary MAS research** (EvoAgent / EvoMAS): selection + mutation/crossover in structured agent-configuration space makes **S** worth testing later, but not trusting blindly — especially when the evolutionary operator or fitness judge is an LLM.

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
