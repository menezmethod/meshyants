# NeuroAI Watch — MESH-107

Cadence: run at a design fork, after a major experiment, or when a notable primary NeuroAI/connectomics paper appears. Do not churn weekly.

This note converts current primary literature into **mechanism → software hypothesis → measurable prediction**. Podcasts and press are leads only. Nothing here is adopted because it is biologically elegant.

**Claim (revised after round-1 evaluators):** this cycle, NeuroAI supplied *experiment-design constraints* and *rejected translations*, not a new coordination primitive. A labeled queue, router, DAG, manager, or Brooks subsumption stack already expresses the v2 software priors. Biology still must not enter without a prediction that can fail.

**Strongest simpler alternative:** keep the existing v2 priors without a literature watch. Round 1’s scheduler critic scored leftover advantage at 0 and was right about A2–A5.

**Falsification of this watch:** the watch treats a podcast as evidence, or promotes a v2 prior to a NeuroAI mechanism without a leftover test vs queue / router / DAG / manager / subsumption.

## Mentat check

**What must be true:** connectomic / NeuroAI regularities are about *information-flow constraints*, not neuron-as-agent identity. Those constraints can be implemented as software topology, signal scope, and budget rules.

**Assumptions that created the question:** v2 currently cites FlyWire, BANC, Cognitive Legos, effort allocation, and evolutionary MAS without primary citations or a what-not-to-copy list. That makes the design unfalsifiable.

**Excluded alternatives:** copying MAP-style LLM modules for every PFC function; treating a global workspace as “brain-like”; evolving phenotypes with an LLM fitness judge before Phase A–C exist.

**Evidence that would change the conclusion:** Experiment A (MESH-103/104) showing modular+sparse MeshyAnts beating all five simpler systems on the heterogeneous mix, or showing it never does.

**Missing evidence:** MeshyAnts has not yet run Experiment A. This watch constrains that experiment; it does not replace it.

**Round-1 finding (accepted):** naming a prediction is too cheap. A2 (bounded verify/repair), A3 (join/fan-out), A4 (skill reuse), and A5 (VoL/bandit) are already scheduler patterns. They are demoted to corroborative priors. S9/S10a/S8b author strings were wrong in round 1 and are corrected against Crossref/arXiv.

---

## Primary sources (this cycle)

Press, interviews, and Brain Inspired transcripts were not used as evidence.

| ID | Source | Kind | Why it is here |
|---|---|---|---|
| S1 | Dorkenwald et al., *Neuronal wiring diagram of an adult brain*, Nature 634:124–138 (2024). DOI [10.1038/s41586-024-07558-y](https://doi.org/10.1038/s41586-024-07558-y) | connectome resource | FlyWire whole-brain map (139,255 neurons, ~5e7 synapses). Dataset, not a coordination proof. |
| S2 | Lin, Yang et al., *Network statistics of the whole-brain connectome of Drosophila*, Nature (2024). DOI [10.1038/s41586-024-07968-y](https://doi.org/10.1038/s41586-024-07968-y) | topology stats | Rich-club (~30% of neurons, degree cutoff 37), 676 broadcasters / 638 integrators (arbitrary 5× in/out rule), reciprocity 0.138, mean directed path 4.42 in the giant SCC, small-worldness high. |
| S3 | Bates, Phelps, Kim, Yang et al., *Distributed control circuits across a brain-and-cord connectome*, Nature (8 Jun 2026). DOI [10.1038/s41586-026-10735-w](https://doi.org/10.1038/s41586-026-10735-w) | brain+cord control | BANC: local sensor–effector loops dominate; AN/DN long-range circuits form behavior-centric modules; learning/navigation regions supervise; authors explicitly analogize to **subsumption / distributed robotic control**. |
| S4 | Tafazoli, Bouchacourt, … Buschman, *Building compositional tasks with shared neural subspaces*, Nature (26 Nov 2025). DOI [10.1038/s41586-025-09805-2](https://doi.org/10.1038/s41586-025-09805-2) | compositionality | Macaque PFC reuses shared subspaces (feature, action) across related tasks and quiets unused ones. Press name: “Cognitive Legos.” |
| S5 | Masís Obando, Musslick, Cohen, *Learning expectations shape cognitive control allocation*, PNAS 122 (2025). DOI [10.1073/pnas.2416720122](https://doi.org/10.1073/pnas.2416720122) | effort / VoL | People spend more control when they expect a task to be learnable, forgoing short-run reward. |
| S6 | Kim, Gu, Park et al., *Capable language models can outgrow the benefits of collaboration*, Nat. Mach. Intell. 8:1157–1172 (24 Jul 2026). DOI [10.1038/s42256-026-01268-y](https://doi.org/10.1038/s42256-026-01268-y) | contradictory MAS | Equal-budget 260-config study: single-agent baseline is the best predictor of whether extra agents help; ~45% capability-saturation rule; sequential tasks: MAS −39% to −70%; independent agents amplify errors 17.2× vs centralized 4.4×. |
| S7 | Webb, Mondal, Momennejad, *A brain-inspired agentic architecture to improve planning with LLMs*, Nat. Commun. 16:8633 (2025). DOI [10.1038/s41467-025-63804-5](https://doi.org/10.1038/s41467-025-63804-5) | rejected translation | MAP: every PFC-inspired function is an LLM prompt module plus an Orchestrator. Useful as a **negative** example (violates Rule 1). |
| S8 | Yuan et al., *EvoAgent*, arXiv:2406.14228 (2024); Hu, Zhang, Trager, Zhang, Yang, Xia, Soatto, *EvoMAS*, arXiv:2602.06511 / ICML 2026 | selection | Structured MAS configuration space can be mutated. EvoMAS uses an **LLM meta-model** as the evolutionary operator and an LLM judge as reward. Defer, do not copy. |
| S9 | Podschun, Betzel, Braun, Markett, *Exploring the Role of the Rich Club in Network Control of Neurocognitive States*, Hum. Brain Mapp. (2026). DOI [10.1002/hbm.70485](https://doi.org/10.1002/hbm.70485) | contradictory topology | In HCP data, prohibiting peripheral nodes from control hurts more than prohibiting the rich club. Rich club fits a **passive data highway**, not a control center. |
| S10 | Mallein, Paparella, Schertzer, Talyigás, *Selection of the fittest or selection of the luckiest*, arXiv:2503.21849 (2025); Karwowski, Hayman, Bai, Kiendlhofer, Griffin, Skalse, *Goodhart’s Law in Reinforcement Learning*, ICLR 2024 (arXiv:2310.09144) | selection hazard | Strong selection on a noisy phenotypic proxy favors luck, not fitness. Proxy optimization past a point lowers true objective. |
| S11 | Dorigo & Stützle, *Ant Colony Optimization*, MIT Press (2004) — secondary classic, used only for the ACO invariant | stigmergy limit | Trace-only search (β→0, no local heuristic η) collapses. Decay is load-bearing. |

Dataset pointers, not arguments: [FlyWire](https://flywire.ai/), [Codex](https://codex.flywire.ai/) (FAFB v783, BANC v888).

### Sources searched and not adopted

- Neuromatch / Allen NeuroAI 2026 meeting (“multi-area interactions”): agenda, not a result.
- BIGMAS (arXiv:2603.15371): “brain-inspired” graph + **central shared workspace + global Orchestrator**. This is a dressed-up manager, not a MeshyAnts mechanism.
- DeLM (arXiv:2606.10662): decentralized claim via shared *verified* context. Keep as a software lead; it is not NeuroAI evidence.
- Structure–function coupling critiques (e.g. BSA 2026 perspective): wiring ≠ computation. Blocks any “the connectome proves this topology computes X” claim.

---

## What this cycle actually adopts

Round 1 treated A1–A7 as “adopted NeuroAI mechanisms.” The scheduler critic rejected that. The map is now three classes. Machine-readable copy: `docs/results/MESH-107/mechanism-map.json` schema v2.

### Q1 — leftover empirical question (not a new primitive)

Local neighborhoods + sparse bridges were already in `ARCHITECTURE.md`. S3’s useful residue is that BANC authors analogize to **subsumption**, so a flat blackboard is a handicapped baseline.

- **Compare:** MeshyAnts local-claim + decaying field + sparse INSIGHT/DANGER/SATURATION versus (i) capability-labeled priority queue, (ii) central load-aware router, (iii) deterministic DAG, (iv) manager-worker, (v) Brooks inhibit hierarchy.
- **Prediction:** On a heterogeneous failure-prone mix, MeshyAnts reduces recovery time or duplicate-claim rate versus all five at matched quality and equal budget. On a sequential DAG it loses or ties the orchestrator.
- **Metrics:** `task_success_rate`, `verified_quality`, `wall_time`, `token_or_tool_cost`, `duplicate_claim_count`, `coordination_message_count`, `error_amplification`, `recovery_time_after_worker_kill`.
- **Falsification:** any of the five simpler systems matches or beats MeshyAnts on quality and coordination cost in the heterogeneous regime.
- **Why the analogy may fail:** fly modules are embodied; software neighborhoods are labels. S6 says extra agents often hurt once a single worker is capable.

### Experiment-design constraints (not runtime features)

| ID | Constraint | Source | Prediction that can fail |
|---|---|---|---|
| C1 | Experiment A includes a DAG lose-regime and a recorded single-worker baseline | S6 | The published spec names both regimes and logs single-worker success first |
| C2 | MESH-103 includes a subsumption/inhibit baseline, not only a flat queue | S3 | Phase A ships that baseline at equal budget |
| C3 | No Queen / LLM-as-fitness until Phase A–C exist; fitness is a vector if it ever exists | S8a/S8b, S10a/S10b | MESH-104/105 have `selection_enabled=false` |
| C4 | No rich-club-controller as a MeshyAnts condition; hub-assign scores as a manager baseline | S2, S9 (Podschun, Betzel, Braun, Markett) | MESH-105 does not treat hub-assign as MeshyAnts |

### Software priors — NeuroAI is not a warrant

These stay in the v2 plan as software. Citing a paper does not make them leftover.

| Was | Prior | Already expressed by |
|---|---|---|
| A2 | Bounded verify/repair | DAG check step; manager retry bound |
| A3 | Join + fan-out | DAG reduce; pub/sub; result log |
| A4 | Versioned composable skills | ARCHITECTURE.md phenotype |
| A5 | Exploration budget | central bandit / ε-greedy |

---

## What not to copy

1. **Neurons are not workers.** A fly neuron has no lease, budget, or phenotype version. Treating 130k neurons as 130k agents is a category error.
2. **Rich club is not a Queen.** S2’s rich club is ~30% of the brain; S9 says it may be a highway. Do not put task assignment in “hub” agents because the fly has hubs.
3. **Do not implement integrator/broadcaster as LLM personas.** Traffic functions are software. Cognition stays in workers (ARCHITECTURE Rule 1).
4. **Do not copy MAP (S7).** “PFC-inspired” modules that are all LLM calls spend model tokens on coordination. That is the failure mode MeshyAnts exists to avoid.
5. **Do not copy BIGMAS / Global Workspace as a requirement.** A globally visible blackboard plus an Orchestrator is a manager-worker system. Calling it brain-inspired does not change the baseline.
6. **Do not hard-code fly neuropils** (mushroom body, GNG, VNC, …) as product packages. Neighborhoods are capability graphs we choose and ablate.
7. **Do not ship pheromone without decay and a local heuristic.** ACO without η collapses; digital traces without decay become litter. v1 `PheromoneRecord` decay is the part of the ant metaphor that has an invariant. The rest is branding.
8. **Do not treat reciprocal motifs as unbounded agent debate.** Bounded verify/repair is a DAG prior (P1), not a FlyWire leftover.
9. **Do not use podcasts or lab-press as evidence.** “Cognitive Legos” is a press name for S4. Brain Inspired can start a search; it cannot close one.
10. **Do not run LLM-as-fitness evolution (S8) as the core claim.** If someone later wants a Queen, fitness must be machine-checked outcomes.
11. **Do not subscribe every worker to the whole field.** That is O(workers × events) global sniffing, which MESH-106 is meant to reject.
12. **Do not infer function from wiring alone.** Connectomes constrain possible flows. They do not prove a software scheduler.

---

## Contradictory evidence (what would make us drop an idea)

| Claim in v2 prose | Tension | Consequence |
|---|---|---|
| “Rich-club / integrators argue against a flat swarm” | S2: rich club is huge; integrator labels are arbitrary. S9: rich club may not drive control. | Keep modular+sparse bridges (A1). Do **not** build a rich-club controller. |
| “Decentralized biologically inspired coordination can outperform” | S6: extra agents often hurt; sequential MAS is a large regression; error amplification is the main risk. | Win condition stays narrow (RESEARCH.md). MESH-101 must include the lose regime. |
| “Pheromone field as ant trail” | ACO without local heuristic fails; stale traces are litter; BANC authors analogize to **subsumption**, not ants. | Field signals need decay + local eligibility. Subsumption-like inhibition is a required baseline (scheduler critic). |
| “Evolve worker phenotypes” | S8 uses LLM judges; S10: strong proxy selection → luck / Goodhart. | Selection stays Phase 4. A6 is a negative-control prediction. |
| “Brain-inspired modules improve planning” (MAP) | S7 spends LLMs on coordination and still looks like a planner with extra prompts. | Reject as a MeshyAnts pattern. Fine as an external baseline later. |

---

## Implications for open tasks (do not start them here)

- **MESH-101:** experiment contract must include DAG-lose and heterogeneous-maybe-win regimes, equal budget, and a single-worker baseline. Record error-amplification, not just success rate.
- **MESH-102:** phenotype = composable versioned skills (A4), not a role enum.
- **MESH-103:** baselines must include manager-worker, central capability router, **and** a subsumption-like inhibit hierarchy (Bates/Brooks), not only a flat queue.
- **MESH-104:** local claim + inhibition; no LLM claim; VoL exploration (A5) as a policy flag.
- **MESH-105:** topologies = flat / modular / modular+integrator-broadcaster / recurrent-verify. Rich-club-controller is a rejected topology.
- **MESH-106:** reject global sniff; count wakeups.
- **Queen / MESH-109 marketplace:** still deferred.

---

## Decision

**Accepted negative result:** this cycle did not extract a new coordination primitive from NeuroAI.

**Keep:** Q1 (Experiment A vs five simpler systems), C1–C4 (contract constraints), R1–R12 (what not to copy), primary-source citations on the five RESEARCH.md priors.

**Demote:** A2–A5 to software priors with no NeuroAI warrant.

**Reject:** neuron-as-agent, rich-club Queen, MAP, BIGMAS-as-MeshyAnts, neuropil packages, pheromone-without-decay, podcast evidence, early LLM-evolution.

Confidence in the *translation discipline* after the demotion: medium.  
Confidence that MeshyAnts will beat simpler systems: **low**, unchanged.

Machine-readable map: [`docs/results/MESH-107/mechanism-map.json`](../results/MESH-107/mechanism-map.json).  
Source list: [`docs/results/MESH-107/sources.json`](../results/MESH-107/sources.json).
