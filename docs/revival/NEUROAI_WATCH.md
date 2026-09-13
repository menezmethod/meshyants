# NeuroAI Watch — MESH-107

Cadence: run at a design fork, after a major experiment, or when a notable primary NeuroAI/connectomics paper appears. Do not churn weekly.

This note converts current primary literature into **mechanism → software hypothesis → measurable prediction**. Podcasts and press are leads only. Nothing here is adopted because it is biologically elegant.

**Claim:** NeuroAI can supply testable coordination mechanisms, but a biological analogy may enter MeshyAnts only when it names a software experiment that can fail.

**Strongest simpler alternative:** keep the existing v2 priors (local neighborhoods, decay, no LLM for routing) without a literature watch. If this note adds no prediction that a priority queue, capability router, DAG, or Brooks-style subsumption stack cannot already express, the watch is ceremony.

**Falsification of this watch:** an adopted idea has no measurable prediction, treats a podcast as evidence, or is already implied by a simpler scheduler with no leftover test.

## Mentat check

**What must be true:** connectomic / NeuroAI regularities are about *information-flow constraints*, not neuron-as-agent identity. Those constraints can be implemented as software topology, signal scope, and budget rules.

**Assumptions that created the question:** v2 currently cites FlyWire, BANC, Cognitive Legos, effort allocation, and evolutionary MAS without primary citations or a what-not-to-copy list. That makes the design unfalsifiable.

**Excluded alternatives:** copying MAP-style LLM modules for every PFC function; treating a global workspace as “brain-like”; evolving phenotypes with an LLM fitness judge before Phase A–C exist.

**Evidence that would change the conclusion:** a controlled equal-budget study showing MeshyAnts never beats a competent router/DAG on the hypothesized win regime, or showing that every “biological” gain is reproduced by a labeled priority queue.

**Missing evidence:** MeshyAnts has not yet run Experiment A. This watch constrains that experiment; it does not replace it.

---

## Primary sources (this cycle)

Press, interviews, and Brain Inspired transcripts were not used as evidence.

| ID | Source | Kind | Why it is here |
|---|---|---|---|
| S1 | Dorkenwald et al., *Neuronal wiring diagram of an adult brain*, Nature 634:124–138 (2024). DOI [10.1038/s41586-024-07558-y](https://doi.org/10.1038/s41586-024-07558-y) | connectome resource | FlyWire whole-brain map (139,255 neurons, ~5e7 synapses). Dataset, not a coordination proof. |
| S2 | Lin, Yang et al., *Network statistics of the whole-brain connectome of Drosophila*, Nature (2024). DOI [10.1038/s41586-024-07968-y](https://doi.org/10.1038/s41586-024-07968-y) | topology stats | Rich-club (~30% of neurons, degree cutoff 37), 676 broadcasters / 638 integrators (arbitrary 5× in/out rule), reciprocity 0.138, mean directed path 4.42 in the giant SCC, small-worldness high. |
| S3 | Bates, Phelps, Kim, Yang et al., *Distributed control circuits across a brain-and-cord connectome*, Nature (8 Jun 2026). DOI [10.1038/s41586-026-10735-w](https://doi.org/10.1038/s41586-026-10735-w) | brain+cord control | BANC: local sensor–effector loops dominate; AN/DN long-range circuits form behavior-centric modules; learning/navigation regions supervise; authors explicitly analogize to **subsumption / distributed robotic control**. |
| S4 | Tafazoli, Bouchacourt, … Buschman, *Building compositional tasks with shared neural subspaces*, Nature (26 Nov 2025). DOI [10.1038/s41586-025-09805-2](https://doi.org/10.1038/s41586-025-09805-2) | compositionality | Macaque PFC reuses shared subspaces (feature, action) across related tasks and quiets unused ones. Press name: “Cognitive Legos.” |
| S5 | Masís, Musslick, Cohen, *Learning expectations shape cognitive control allocation*, PNAS 122 (2025). DOI [10.1073/pnas.2416720122](https://doi.org/10.1073/pnas.2416720122) | effort / VoL | People spend more control when they expect a task to be learnable, forgoing short-run reward. |
| S6 | Kim, Gu, Park et al., *Capable language models can outgrow the benefits of collaboration*, Nat. Mach. Intell. 8:1157–1172 (24 Jul 2026). DOI [10.1038/s42256-026-01268-y](https://doi.org/10.1038/s42256-026-01268-y) | contradictory MAS | Equal-budget 260-config study: single-agent baseline is the best predictor of whether extra agents help; ~45% capability-saturation rule; sequential tasks: MAS −39% to −70%; independent agents amplify errors 17.2× vs centralized 4.4×. |
| S7 | Webb, Mondal, Momennejad, *A brain-inspired agentic architecture to improve planning with LLMs*, Nat. Commun. 16:8633 (2025). DOI [10.1038/s41467-025-63804-5](https://doi.org/10.1038/s41467-025-63804-5) | rejected translation | MAP: every PFC-inspired function is an LLM prompt module plus an Orchestrator. Useful as a **negative** example (violates Rule 1). |
| S8 | Yuan et al., *EvoAgent*, arXiv:2406.14228 (2024); Amazon Science, *EvoMAS*, arXiv:2602.06511 / ICML 2026 | selection | Structured MAS configuration space can be mutated. EvoMAS uses an **LLM meta-model** as the evolutionary operator and an LLM judge as reward. Defer, do not copy. |
| S9 | Karakoc et al., *Exploring the Role of the Rich Club in Network Control of Neurocognitive States*, Hum. Brain Mapp. (2026). DOI [10.1002/hbm.70485](https://doi.org/10.1002/hbm.70485) | contradictory topology | In HCP data, prohibiting peripheral nodes from control hurts more than prohibiting the rich club. Rich club fits a **passive data highway**, not a control center. |
| S10 | Karalis, Slijepcevic, et al., *Selection of the fittest or selection of the luckiest*, arXiv:2503.21849 (2025); Karwowski, Hayman, et al., *Goodhart’s Law in Reinforcement Learning*, ICLR 2024 | selection hazard | Strong selection on a noisy phenotypic proxy favors luck, not fitness. Proxy optimization past a point lowers true objective. |
| S11 | Dorigo & Stützle, *Ant Colony Optimization*, MIT Press (2004) — secondary classic, used only for the ACO invariant | stigmergy limit | Trace-only search (β→0, no local heuristic η) collapses. Decay is load-bearing. |

Dataset pointers, not arguments: [FlyWire](https://flywire.ai/), [Codex](https://codex.flywire.ai/) (FAFB v783, BANC v888).

### Sources searched and not adopted

- Neuromatch / Allen NeuroAI 2026 meeting (“multi-area interactions”): agenda, not a result.
- BIGMAS (arXiv:2603.15371): “brain-inspired” graph + **central shared workspace + global Orchestrator**. This is a dressed-up manager, not a MeshyAnts mechanism.
- DeLM (arXiv:2606.10662): decentralized claim via shared *verified* context. Keep as a software lead; it is not NeuroAI evidence.
- Structure–function coupling critiques (e.g. BSA 2026 perspective): wiring ≠ computation. Blocks any “the connectome proves this topology computes X” claim.

---

## Adopted ideas

Each idea below is **adopted only as a software hypothesis**. Confidence is for the translation, not for MeshyAnts winning.

### A1 — Local modules + sparse long-range bridges

- **Source:** S3 (primary); S1/S2 as anatomy.
- **Mechanism:** Effectors are mostly driven by same-body-part sensors (local loops). AN/DN cells form behavior-centric modules. Higher regions supervise; they are not required for every action. Bates et al. compare this to distributed / subsumption control, not to a pheromone swarm.
- **Software hypothesis:** Workers subscribe to a capability neighborhood. Cross-neighborhood traffic is sparse (INSIGHT / DANGER / SATURATION), not a global sniff of every TODO.
- **Measurable prediction:** On a heterogeneous, failure-prone mix (MESH-104/105), local+bridge topology cuts coordination messages and duplicate claims versus a flat global blackboard at matched quality. On a sequential DAG (SonicTale-like), it does **not** beat a deterministic orchestrator on quality or wall time.
- **Falsification:** Central router or DAG matches or beats local+bridge on *both* quality and coordination cost in the heterogeneous regime.
- **Why the analogy may fail:** BANC “modules” are sensorimotor body-part circuits in one animal. Software “neighborhoods” are capability labels we invent. Locality in the fly is spatial/embodied; ours is semantic. A subsumption stack with hardcoded inhibit edges may capture the same gain.

### A2 — Recurrence as bounded verify/repair, not chat

- **Source:** S2.
- **Mechanism:** Reciprocity is over-represented (p=0.138). ~2/3 of neurons participate in at least one reciprocal pair. Reciprocal edges are enriched for excitatory–inhibitory pairs (ACh–GABA, ACh–Glu).
- **Software hypothesis:** A topology edge may send work to a verifier and, on fail, to a repair worker, with a retry bound. Software does the loop. No LLM is used to decide “whether to recur.”
- **Measurable prediction:** A bounded verifier→repair recurrence lowers undetected-error rate versus one-shot workers at a measured extra token/time cost. Removing the bound (unbounded debate) raises cost without a matching quality gain.
- **Falsification:** A single terminal checker (no recurrence) matches error rate at lower cost; or unbounded recurrence is required for the gain.
- **Why the analogy may fail:** Synaptic reciprocity is millisecond local feedback, not a multi-agent review cycle. Over-represented motifs are not a warrant for more agents talking.

### A3 — Integrator / broadcaster as traffic roles, not Queens

- **Source:** S2; contradiction S9.
- **Mechanism:** Lin et al. label 676 broadcasters (out ≥ 5× in) and 638 integrators (in ≥ 5× out). They call the cutoff **arbitrary**. 37,093 remaining rich-club neurons are balanced. The fly rich club is ~30% of neurons, not a tiny elite; an interneuropil-constrained null model accounts for the rich-club excess. S9: rich-club nodes look more like a highway than a controller.
- **Software hypothesis:** An integrator writes a combined RESULT/INSIGHT from several neighborhoods. A broadcaster fans out a high-confidence DANGER/INSIGHT. Neither assigns tasks. Queen/selection stays Phase 4.
- **Measurable prediction:** Adding a small, fixed set of integrator/broadcaster nodes reduces missed cross-module constraint failures versus modular-only wiring, while **task assignment remains local-claim** (no increase in central assign rate). Ablating them raises missed-constraint failures. Replacing them with a manager that assigns work is a different (simpler) system and must be scored as a baseline, not as MeshyAnts.
- **Falsification:** A global result log, or a manager tree, matches the missed-constraint metric at equal or lower cost.
- **Why the analogy may fail:** Degree imbalance ≠ computational role. S9 directly weakens “put control in the rich club.” If integrators become hidden managers, the experiment is confounded.

### A4 — Compositional phenotypes, not named roles

- **Source:** S4.
- **Mechanism:** Shared PFC subspaces encode reusable task components (e.g. color category, action axis) and are engaged in sequence; unused subspaces are quieted.
- **Software hypothesis:** A worker is `model/tool + policy + versioned skills + permissions + context strategy + budget`. Skills recombine. Unused skills stay out of the prompt. If any of those changes materially, version the phenotype (MESH-102).
- **Measurable prediction:** On a family of compositionally related tasks, a skill-reusing phenotype has lower cold-start tokens and no worse quality than a pile of role-specialized prompts that restate overlapping instructions. Forcing unused skills into context increases tokens without quality.
- **Falsification:** Monolithic role prompts match reuse on quality and cost.
- **Why the analogy may fail:** S4 is within-brain geometry in two monkeys on three lab tasks. It is not a result about multi-agent role design. Compositional *representations* ≠ compositional *workers*.

### A5 — Spend extra budget where information gain is high

- **Source:** S5; software-only support from S6’s “when not to add agents.”
- **Mechanism:** Control allocation is not myopic reward maximization. Expected learnability predicts later willingness to deliberate.
- **Software hypothesis:** Exploration / extra attempts are allocated by a deterministic value-of-learning rule (e.g. uncertainty × payoff × remaining budget), never by an LLM “trying harder.” On saturated easy tasks, do not add agents (S6).
- **Measurable prediction:** After a new cold-start capability appears, VoL reaches a target success rate in fewer subsequent tasks than greedy exploit-only, at higher short-run cost and lower long-run regret. On a static easy workload, VoL wastes budget versus greedy.
- **Falsification:** ε-greedy or a central contextual bandit matches the adaptation curve at equal or lower cost.
- **Why the analogy may fail:** Human deliberation time ≠ agent replica count. S5 is one perceptual task. S6 says extra agents often hurt once a single capable worker exists.

### A6 — Selection is deferred and multi-objective

- **Source:** S8, S10.
- **Mechanism:** Evolving MAS configs can improve some benchmarks, but published methods use LLM judges and LLM meta-operators. Strong selection on a noisy proxy selects luck (S10) and Goodharts (S10 / ICLR 2024).
- **Software hypothesis:** Do not run Queen / phenotype mutation until Phase A–C have a working harness. Fitness, when it exists, is a vector: quality, verification confidence, cost, latency, reliability. Never tasks/hour alone. Do not use an LLM as the evolutionary operator for the core experiment.
- **Measurable prediction (negative control):** Introducing single-proxy selection before Experiment A produces higher throughput and **lower** verified quality than no-selection.
- **Falsification:** Early single-proxy selection improves verified quality without a hidden quality leak. If that happens, this warning is wrong — record it.
- **Why the analogy may fail:** Biological fitness is reproductive. Software “fitness” is whatever we log. That gap is the Goodhart surface.

### A7 — Do-not-swarm gate (capability saturation)

- **Source:** S6. This is **MAS evidence**, not biology. Included because it is the strongest current constraint on when a NeuroAI-inspired swarm is even allowed to claim a win.
- **Mechanism:** Under equal tools/prompts/budget, extra agents help mainly when the single-agent baseline is weak and the task is parallelizable or dynamically explorative. Sequential reasoning and high single-agent skill are lose regimes. Architectures without a verification bottleneck amplify errors.
- **Software hypothesis:** The experiment contract (MESH-101) must include (i) a sequential DAG regime expected to favor manager/DAG, (ii) a heterogeneous failure-prone regime hypothesized to favor MeshyAnts, and (iii) a recorded single-worker baseline. If that baseline is already strong, MeshyAnts should be predicted to lose.
- **Measurable prediction:** MeshyAnts loses or ties the DAG regime on quality and wall time. Any claimed win is confined to the heterogeneous regime and must survive ablation of topology/feedback. Unverified volunteer swarms show higher error amplification than the central router.
- **Falsification:** MeshyAnts wins *universally* (treat as suspicious / likely baseline handicap) or never wins even on the hypothesized regime (MeshyAnts claim fails).
- **Why the analogy may fail:** S6’s MAS variants are LLM conversation topologies, not MeshyAnts’ software field. Their “decentralized” condition may be weaker than a well-engineered local-claim system — or stronger. Only the equal-budget harness can tell.

---

## What not to copy

1. **Neurons are not workers.** A fly neuron has no lease, budget, or phenotype version. Treating 130k neurons as 130k agents is a category error.
2. **Rich club is not a Queen.** S2’s rich club is ~30% of the brain; S9 says it may be a highway. Do not put task assignment in “hub” agents because the fly has hubs.
3. **Do not implement integrator/broadcaster as LLM personas.** Traffic functions are software. Cognition stays in workers (ARCHITECTURE Rule 1).
4. **Do not copy MAP (S7).** “PFC-inspired” modules that are all LLM calls spend model tokens on coordination. That is the failure mode MeshyAnts exists to avoid.
5. **Do not copy BIGMAS / Global Workspace as a requirement.** A globally visible blackboard plus an Orchestrator is a manager-worker system. Calling it brain-inspired does not change the baseline.
6. **Do not hard-code fly neuropils** (mushroom body, GNG, VNC, …) as product packages. Neighborhoods are capability graphs we choose and ablate.
7. **Do not ship pheromone without decay and a local heuristic.** ACO without η collapses; digital traces without decay become litter. v1 `PheromoneRecord` decay is the part of the ant metaphor that has an invariant. The rest is branding.
8. **Do not treat reciprocal motifs as unbounded agent debate.** Recurrence is A2 (bounded verify/repair) or it is slop.
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

**Adopt A1–A7 as constraints and predictions, not as implemented features.**  
**Reject** neuron-as-agent, rich-club Queen, MAP, BIGMAS-as-MeshyAnts, neuropil packages, pheromone-without-decay, podcast evidence, early LLM-evolution.

Confidence in the *translation discipline*: medium-high.  
Confidence that MeshyAnts will beat simpler systems: **low**, unchanged. S3 and S6 both say the plausible win is narrow.

Machine-readable map: [`docs/results/MESH-107/mechanism-map.json`](../results/MESH-107/mechanism-map.json).  
Source list: [`docs/results/MESH-107/sources.json`](../results/MESH-107/sources.json).
