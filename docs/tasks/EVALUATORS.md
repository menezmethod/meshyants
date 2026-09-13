# Evaluator Gauntlet

Material work does not advance only because tests pass.

## Evaluation protocol

Use a **fresh context** for each material evaluator round when practical. The judge should receive:
- the hypothesis and falsification condition
- current evidence/artifacts
- baseline results
- previous round's score/verdict
- relevant constraints/budget

Do **not** preload the builder's chain of reasoning or advocacy. The evaluator should be able to disagree with the target itself.

## Stable scorecard

Score each dimension 0–5. Keep the rubric stable across rounds.

| Dimension | 0 | 5 |
|---|---|---|
| Scientific validity | unsupported / confounded | strong evidence, falsifiable, repeatable |
| Advantage over simpler baseline | no advantage | clear, reproducible meaningful advantage |
| Performance / cost | wasteful / unknown | measured and competitive |
| Resilience / scalability | brittle / unmeasured | survives relevant perturbations and scales credibly |
| Code / experiment quality | slop / irreproducible | minimal, clear, tested, reproducible |

**Total: 0–25.** Never hide a weak dimension behind a strong total.

## 1. Scientific skeptic

Ask:
- What must be true for this result to support the claim?
- What simpler explanation fits the same data?
- Was the baseline handicapped?
- Would this survive repeated runs / seeds?
- What evidence would reverse the conclusion?

**Can reject:** unsupported research claims, bad metrics, wrong experiments, or benchmark conclusions.

## 2. Scheduler critic

Assume MeshyAnts is unnecessary.

Try to reproduce the gain with:
- priority queue + capability labels
- central load-aware router
- deterministic DAG/workflow
- manager → worker orchestration

If the simpler system wins or ties within noise/cost, record that result.

## 3. Performance/scalability critic

Measure when relevant:
- wall time
- tokens / model calls / dollars
- worker wakeups and failed claims
- event amplification
- memory/CPU/network overhead

Reject hidden O(N×events) global sniffing presented as scalable.

## 4. Failure/security critic

Kill workers, expire leases, replay messages, inject stale owners, duplicate tasks and malformed/untrusted capability ads. External effects must be idempotent or fenced.

## 5. Code/slop evaluator

Reject:
- abstractions with no experiment needing them
- duplicated framework code
- TODO-driven fake completeness
- tests that only mirror implementation
- AI-generated prose/code that cannot explain its invariant

Prefer deletion over architecture that has not earned its existence.

## Round verdict

Return:
- **Verdict:** PASS / PASS WITH RISKS / REJECT
- **Score:** each dimension + total /25
- **Delta:** change from prior round, with regressions called out
- **Evidence:** concrete observations
- **Strongest objection**
- **Repeated blocker:** yes/no + what
- **Target challenge:** is the hypothesis/metric/benchmark itself wrong?
- **Next experiment/fix:** one highest-value next move

No evaluator may approve based on aesthetics, novelty, or amount of code.
