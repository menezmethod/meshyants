# V2 Experiment Contract

**Status:** locked for Phase A design (MESH-101).  
**Contract version:** `1`  
**Machine check:** `go run ./cmd/expcontract validate <run.json>`

This document is the only place an agent may take comparison rules from. If a rule is not here or in the versioned schema, it is not a rule.

## Mentat record

**Claim:** A four-arm experiment is comparable iff the task set, worker pool, tools, models, oracles, seeds, and budget envelope are frozen, and only the allocation policy varies.

**Necessary assumptions:** Phase A can isolate allocation from topology/learning/selection. Deterministic synthetic tasks can expose allocation differences. MESH-102 will define phenotype/lease semantics before exclusive-effect workloads run.

**Strongest alternative:** Leave Phase A as prose in `docs/revival/EXPERIMENTS.md` and let the harness invent metrics.

**Evidence for:** Informal docs already disagree on how many systems Phase A compares (four in MESH-101 vs five in EXPERIMENTS.md) and do not define denominators, budget equality, or what "win" means.

**Evidence against:** A schema can still be gamed by a weak single-agent phenotype or a workload that secretly favors one arm.

**Missing evidence:** No harness results yet (MESH-103). No locked phenotype schema yet (MESH-102).

**Falsification test:** The validator accepts a handicapped spec, or another agent still has to invent a metric, budget rule, or verdict rule to run Phase A.

**Decision / confidence:** Lock a small machine-checked contract now. Medium-high confidence this prevents unfalsifiable framework-building; low confidence the first synthetic workload predicts real user value.

## What this contract is not

- Not a MeshyAnts allocator, topology, or reputation design.
- Not a worker phenotype / lease / checkpoint spec (MESH-102).
- Not a claim that MeshyAnts will win.
- Not permission to treat SonicTale / other DAG workflows as the decisive swarm benchmark.

## Independent variable

**Allocation policy.** Everything else below is held constant across arms.

## Required systems

A Phase A run spec MUST include these four arms:

| ID | Meaning | Worker materialization |
|---|---|---|
| `single_agent` | One worker, no manager, no router, no field | Exactly one instance of `single_agent_phenotype_id` |
| `manager_worker` | Deterministic manager assigns each ready task to a worker | Full `worker_pool` |
| `central_router` | Capability-matched central router (v1 `routing.Router` is the starting point, not a free pass) | Full `worker_pool` |
| `meshyants` | Local eligibility + thresholds + contextual reputation + inhibition + exploration (MESH-104) | Full `worker_pool` |

Optional fifth arm: `flat_blackboard` (volunteer claims, no reputation/inhibition). Allowed because EXPERIMENTS.md lists it. Not required to lock MESH-101.

**Allocation-group fairness:** `manager_worker`, `central_router`, `meshyants`, and `flat_blackboard` MUST use the identical `worker_pool`. Different pool size or secret extra phenotypes is a handicapped spec and is invalid.

**Emergence-group meaning:** `single_agent` is the best-individual baseline for `emergence_delta = colony_quality - single_agent_quality`. Parallelism of the colony vs one worker is **not** evidence that MeshyAnts beat a router.

## Workload classes

| Class | Use | Decisive for swarm advantage? |
|---|---|---|
| `homogeneous_dag` | Known graph, known skills | No. Simpler systems are expected to tie or win. A MeshyAnts win here is a baseline-handicap smell. |
| `heterogeneous_uncertain` | Mixed skills, uncertain expertise, optional follow-up work | Yes, for Phase A allocation. |
| `perturbation` | Worker death, cold start, workload shift | Yes, for Phase C adaptation — not Phase A. |
| `durable_dag` | SonicTale-like write → TTS → finalize | No. Durability/failure only. Setting `decisive_swarm_benchmark: true` on this class is invalid. |

The first locked fixture is `phase-a-synthetic-v1` (`heterogeneous_uncertain`).

## Benchmark task schema

Each task MUST provide:

- `id` (unique in the set)
- `class` (one of the four workload classes)
- `required_capabilities` (string labels; empty means any worker may attempt)
- `arrival`: `initial` or `{ "on_success_of": "<task_id>" }` or `{ "on_failure_of": "<task_id>" }`
- `exclusive_side_effect` (bool)
- `timeout_ms`, `max_attempts` (≥ 1)
- `payload` or `payload_ref`
- `verifier` with a closed `kind`
- `quality_oracle` with a closed `kind`

Closed verifier kinds:

| Kind | Pass rule |
|---|---|
| `deterministic_json_equal` | Canonical JSON of output equals `expected` / `expected_ref` |
| `deterministic_command` | Command exits 0; optional stdout match |
| `rubric_score` | Documented rubric file returns `[0,1]` |
| `human_label` | Pre-existing labels file; labels MAY NOT be invented at run time |

Closed quality-oracle kinds:

| Kind | Score |
|---|---|
| `binary_from_verifier` | `1` on pass, `0` on fail/timeout |
| `rubric_score` | Same `[0,1]` as the rubric verifier |
| `explicit_field` | Named numeric field already on the output, clamped to `[0,1]` |

`exclusive_side_effect: true` is forbidden until the run names a `lease_contract_ref` that exists. Phase A synthetic uses none.

## Success denominator

Let `T_triggered` be every task whose arrival condition has occurred (`initial` tasks always have).

- Denominator for `task_success_rate` and mean `quality` = `|T_triggered|`
- A conditional task that never triggers is neither success nor failure
- Multiple attempts at one task count as one task: success if any **accepted** result lands before timeout
- An accepted result on an exclusive task requires exactly-once external effect (MESH-102). Until then, exclusive tasks are invalid.

## Budget envelope

One envelope, copied to every arm. Per-arm budget overrides are invalid.

| Field | Unit | Rule |
|---|---|---|
| `max_wall_ms` | ms | Hard cap. Clock starts at first allocation decision, ends at last accepted artifact or timeout. |
| `max_model_calls` | count | All model calls, including retries and coordination. |
| `max_input_tokens` | tokens | Same tokenizer for every arm (`tokenizer` field). |
| `max_output_tokens` | tokens | Same tokenizer. |
| `max_cost_micros` | USD × 1e6 | Provider list price recorded in the run spec `price_book`, or `0` when `models` is empty. |
| `max_worker_wakeups` | count | A wakeup is a worker leaving idle to inspect or execute work. |
| `max_exclusive_lease_attempts` | count | Meaningful after MESH-102. |

Rules:

1. Exceeding any hard cap marks the **run** `budget_exceeded`. That run cannot be used as a quality win. It still counts in the repeat set (as a failure).
2. Unused budget is not a penalty. Efficiency is `cost_micros`, `wall_time_ms`, and coordination counters.
3. The single-agent arm gets the **same** envelope, not `1/N`.
4. Baselines MUST NOT be forced to spend dummy LLM calls. Software coordination is free in the model-call counter.
5. `meshyants` coordination MUST be software. Any `coordination_model_calls > 0` on that arm is a **protocol violation** (invalid run), not merely expensive. This encodes Architecture Rule 1: AI does cognition; software does coordination.
6. Duplicate work spends budget. It does not add extra successes.
7. If `models` is empty, token and USD caps MAY be `0`. If `models` is non-empty, those caps MUST be > 0 and `tokenizer` plus `price_book` are required.

## Metrics

Primary (must be reported):

| ID | Definition |
|---|---|
| `task_success_rate` | `passes / |T_triggered|` |
| `quality` | Mean of per-task quality-oracle scores over `T_triggered` |
| `wall_time_ms` | Envelope clock |
| `cost_micros` | Summed model USD |
| `model_calls` | Summed model calls |
| `input_tokens`, `output_tokens` | Summed tokens |

Secondary (required before any MeshyAnts advantage claim):

| ID | Definition |
|---|---|
| `duplicate_work_rate` | Extra executions of a task that already has an accepted result or an exclusive lease held by someone else, divided by total executions |
| `wakeup_count` | Worker wakeups |
| `failed_claim_count` | Claim attempts that lost the race or were ineligible |
| `allocation_message_count` | Messages used to assign/claim/inhibit |
| `coordination_model_calls` | Model calls whose purpose was assignment, claiming, lease, retry, or routing math |
| `recovery_time_ms` | Failure-injection instant → first subsequent accepted result. Null if no injection. |
| `emergence_delta` | `quality(meshyants) - quality(single_agent)` on the same seed. Reported; **not** a standalone win condition. |

Do not collapse these into one fitness scalar. Phase D selection, if it happens, must stay multi-objective.

## Comparison groups and win rule

**Allocation group** (`manager_worker`, `central_router`, `meshyants`, optional `flat_blackboard`): same pool. Primary metrics: success, quality, cost, wall time, model calls. This is the group that can support or falsify a Phase A allocation claim.

**Emergence group** (`single_agent`, `meshyants`): primary metrics: success and quality only. Wall time here is informational.

Repeats:

- `repeats ≥ 3` for any recorded Phase A verdict; ≥ 5 before publication
- `seeds` length equals `repeats`, all unique
- Seed `i` is the sole entropy source for stochastic allocation on repeat `i`
- RNG: Go `math/rand/v2` PCG with that seed, or a documented equivalent named in `rng_algorithm`

Win on one metric, one pair of arms, same group:

1. Compute per-seed paired differences.
2. Significant if the mean difference exceeds the noise threshold **and** at least `ceil(0.8 * repeats)` paired seeds have the same sign (or a bootstrap 95% CI excludes 0 — either rule, named in the run spec).
3. Otherwise `tie`.

Default noise thresholds:

| Metric | Threshold |
|---|---|
| `task_success_rate`, `quality` | 0.05 absolute |
| `wall_time_ms`, `cost_micros`, tokens, model calls | 10% of the better arm's mean |
| `duplicate_work_rate` | 0.05 absolute |

Verdict vocabulary (allocation group, Phase A):

| Verdict | Meaning |
|---|---|
| `meshyants_advantage` | MeshyAnts is significant-better on ≥1 primary metric and not significant-worse on any other primary metric |
| `falsified` | `central_router` or `manager_worker` matches or beats MeshyAnts on **all** primary metrics (ties count as matches) |
| `mixed` | Each side is significant-better on at least one primary metric |
| `inconclusive` | No significant differences, or repeats below the minimum |

A MeshyAnts wall-time win that is only vs `single_agent` is `inconclusive` for allocation.

## Falsification (project-level)

From `docs/revival/RESEARCH.md`, restated so it can be applied:

MeshyAnts is unsuccessful if, under this contract's equal envelopes, `manager_worker` or `central_router` consistently records `falsified` or non-MeshyAnts advantage across the **decisive** classes (`heterogeneous_uncertain` and later `perturbation`).

A win on `homogeneous_dag` or `durable_dag` alone does not save the project claim.

**Emergence is forbidden** unless the run shows: both groups, identical envelopes, costs and latency, at least one failure or negative case, an allocation trace, and an ablation that removes the claimed mechanism.

## Ablation rule (for later claims)

Any claimed mechanism (reputation, inhibition, exploration, topology) MUST have a named ablation arm that keeps the same pool/budget/tasks and disables only that mechanism. If the ablation matches the full system within noise, the mechanism is rejected.

## Mapping onto v1 substrate

Reuse, do not re-encode:

- Work items travel as signed `TaskAtom`s
- Workers advertise with `CapabilityAdvertisement`
- Outcomes travel as `PheromoneRecord` (`TODO`, `INSIGHT`, `DANGER`, `SAFE`) with decay
- `ReputationEvent` is allowed as a slow signal; it is not a Phase A requirement
- Existing `routing.Router` is the seed of `central_router`, not automatically a competent baseline — MESH-103 must give it the same capability labels the other arms see

`requirements_json` on a `TaskAtom` SHOULD carry `required_capabilities` from the task spec so v1 routing can participate without a parallel task type.

## What MESH-103 must emit

Each arm-repeat writes one result object matching `schema/run-result.schema.json`: `run_id`, `system`, `seed`, `git_sha`, `budget_used`, `budget_exceeded`, `protocol_violation`, metrics, per-task rows, `trace_ref`.

A comparison object matching `schema/comparison.schema.json` is required before a verdict may be recorded.

## First fixture

`docs/experiments/examples/phase-a-synthetic/` is the worked example: 12 deterministic tasks, mixed capabilities, one follow-up spawn, no models, no exclusive effects. Another agent should be able to implement a harness against those three JSON files without adding rules.
