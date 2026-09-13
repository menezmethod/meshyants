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

**Missing evidence:** No harness results yet (MESH-103). No locked phenotype schema yet (MESH-102). Reputation/claiming is MESH-104; this contract only locks what that allocator may see.

**Falsification test:** The validator accepts a handicapped spec, or another agent still has to invent tool semantics, baseline assignment, mismatch behavior, a metric, a budget rule, or verdict math.

**Decision / confidence:** Lock a machine-checked contract plus two fixtures. Calibration DAG is not decisive. Medium confidence this stops invented rules; still low confidence the uncertain fixture predicts real user work.

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
| `manager_worker` | Central queue + **AssignStaticRetry** | Full `worker_pool` |
| `central_router` | Same **AssignStaticRetry** math; different control locus only | Full `worker_pool` |
| `meshyants` | May use outcomes beyond retry. Must not read `true_capabilities`. | Full `worker_pool` |

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

Two locked fixtures:

| Fixture | Class | Decisive? | Role |
|---|---|---|---|
| `phase-a-synthetic-v1` | `homogeneous_dag` | no | Harness calibration. Competent static routers should score success=1. A MeshyAnts win on success/quality here is a bug smell. |
| `phase-a-uncertain-v1` | `heterogeneous_uncertain` | yes | Phase A claim. ≥25% of tasks have empty or wrong *visible* labels. |

`decisive_swarm_benchmark: true` is invalid on `homogeneous_dag`, `durable_dag`, or any task set with `uncertain_fraction < 0.25`.

## Benchmark task schema

Each task MUST provide:

- `id` (unique in the set)
- `class` (one of the four workload classes)
- `required_capabilities` (visible labels; empty means the static baseline may assign any idle worker)
- `true_capabilities` (execution requirement; omitted means equal to `required_capabilities`). **Allocators must not read this field.**
- `tool` (closed kind; see Tools)
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
| `max_allocation_messages` | count | Assign/claim/inhibit messages. |

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
2. Significant if the mean difference exceeds the noise threshold **and** at least `ceil(0.8 * repeats)` paired seeds have the same sign. `|diff| <= noise` is a tie, including float error on the threshold.
3. Otherwise `tie`.
4. The only win rule is `paired_sign_80`.

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
| `falsified` | A simpler arm is tied-or-better on every primary and significant-better on at least one; **or** the fixture is decisive and MeshyAnts has no significant primary advantage (all-tie included) |
| `mixed` | Each side is significant-better on at least one primary metric |
| `inconclusive` | No significant differences on a **non-decisive** fixture, protocol violation, or repeats below the minimum |

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
- Existing `routing.Router` is **not** the Phase A baseline. Phase A `central_router` and `manager_worker` MUST call `AssignStaticRetry` / `RunStaticRetry`. v1 `routing.Router` may be used only if it implements that policy on the same visible labels.

`requirements_json` on a `TaskAtom` SHOULD carry visible `required_capabilities` only, never `true_capabilities`.

## What MESH-103 must emit

Each arm-repeat writes one result object matching `schema/run-result.schema.json`: `run_id`, `system`, `seed`, `git_sha`, `budget_used`, `budget_exceeded`, `protocol_violation`, metrics, per-task rows, `trace_ref`.

A comparison object matching `schema/comparison.schema.json` is required before a verdict may be recorded.

## Tools (locked)

A capable worker MUST implement these exact functions (`internal/expcontract.Execute`). A worker missing `true_capabilities` MUST return `{"error":"capability_mismatch"}` and fail the verifier.

| Tool | Input | Output |
|---|---|---|
| `json.add_count` | `{items: [...]}` | `{items, n: len(items)}` |
| `json.pick_keys` | `{keep: [k...], ...}` | object of named keys |
| `json.merge` | `{left, right}` | shallow merge, right wins |
| `json.sort_keys` | `{keys: [..]}` | `{keys}` sorted ascending |
| `json.unwrap_msg` | `{raw: {msg}}` | `{text: raw.msg}` |
| `json.wrap_text` | any | `{wrapped: true, n: 1}` |
| `text.classify` | `{text}` | `{label}`: `infra` if text contains `disk` or `/var`; `auth` if `token` or `user`; `billing` if `invoice` or `unpaid`; else `other` |

`effective_exec_ms = task.simulate_exec_ms + worker.simulate_exec_ms`.

These fixtures use `binary_from_verifier`, so `quality == task_success_rate`. That is honest, not a second signal. Do not invent a quality rubric.

## AssignStaticRetry (locked baselines)

`manager_worker` and `central_router` share `internal/expcontract.AssignStaticRetry` / `RunStaticRetry`. They are not a no-retry strawman.

1. Among idle `VisibleEligible` workers, minus instance IDs that already failed this task: fewest capabilities, then lowest `simulate_exec_ms`, then `instance_id`.
2. On `capability_mismatch` or verify fail, exclude that instance and pick again, up to `max_attempts`.
3. If none eligible, the task stays queued. Do not assign a worker that is visibly ineligible.
4. `SimulateStaticRetry` is the locked success/quality those arms must reproduce on these fixtures. `MakespanMS` is the locked `wall_time_ms` clock (per-worker busy time, not process wall clock).
5. `AssignStatic` without exclude is only an ablation, not a required arm.

`meshyants` MUST NOT read `true_capabilities`. It may use outcomes in additional ways (reputation, inhibition). On `phase-a-uncertain-v1`, retry-only is expected to reach the same success/quality as a competent MeshyAnts — that is `falsified`, not a MeshyAnts win. A later fixture must separate reputation from retry.

`single_agent` always uses one instance of `single_agent_phenotype_id`.

`Verify` is `CanonicalJSON` equality plus `binary_from_verifier`. `Triggered` is the success-denominator helper. Do not invent another equality or arrival rule.

## Failure injection

Contract v1 allows only `failure_injection: null` or `{"kind":"none"}`. Other kinds are reserved. `lease_contract_ref`, if set, must parse as `{kind: exclusive_fence, fencing: token, exactly_once: true}`. A task set or sibling JSON file is not a lease.

## Fixtures

- `docs/experiments/examples/phase-a-synthetic/` — labeled calibration DAG
- `docs/experiments/examples/phase-a-uncertain/` — decisive Phase A fixture (5/12 tasks have empty or wrong visible labels)

`go run ./cmd/expcontract validate` on either `run.json`. Import `Execute`, `Verify`, `Triggered`, `AssignStaticRetry`, `RunStaticRetry`, and `MakespanMS`.
