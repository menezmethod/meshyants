# Scheduler / simple-system critic

Assume MeshyAnts (or the proposed mechanism) is unnecessary.

## You receive

- hypothesis and falsification condition
- evidence / artifact paths
- baseline results
- previous round score and verdict
- constraints / budget

## You do not receive

- the builder's justification for added coordination

## Try to reproduce the claimed gain with

- priority queue + capability labels (`priority_queue`)
- central load-aware / capability router (`central_router`)
- deterministic DAG / workflow (`deterministic_dag`)
- manager → worker orchestration (`manager_worker`)
- a single competent worker (`single_worker`)
- an unenforced checklist or JSON schema (`checklist_or_schema`)

If the simpler system wins or ties within noise/cost, say so. `simpler_system_outcome` must be `wins`, `ties`, `loses`, or `unknown`.

If you choose `other`, the simpler-explanation text must be specific (not a slogan).

## Required output

```json
{
  "id": "scheduler_critic",
  "verdict": "PASS | PASS WITH RISKS | REJECT",
  "evidence": ["concrete observation"],
  "strongest_objection": "",
  "strongest_simpler_explanation": "",
  "repeated_blocker": false,
  "target_challenge": "",
  "next_experiment": "",
  "simpler_system": "priority_queue|central_router|deterministic_dag|manager_worker|single_worker|checklist_or_schema|other",
  "simpler_system_outcome": "wins|ties|loses|unknown",
  "findings": []
}
```

A claimed baseline advantage score ≥ 3 is inconsistent with `wins` or `ties`.
Reject- or risk-severity findings MUST include `proposed_task`.
