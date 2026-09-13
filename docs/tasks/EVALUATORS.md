# Evaluator Gauntlet

Material work does not advance only because tests pass. Use independent evaluator passes.

## 1. Scientific skeptic

Ask:
- What must be true for this result to support the claim?
- What simpler explanation fits the same data?
- Was the baseline handicapped?
- Would this survive repeated runs / seeds?
- What evidence would reverse the conclusion?

**Can reject:** unsupported research claims or benchmark conclusions.

## 2. Scheduler critic

Assume MeshyAnts is unnecessary.

Try to reproduce the gain with:
- priority queue + capability labels
- central load-aware router
- deterministic DAG/workflow
- manager → worker orchestration

If the simpler system wins, record that result.

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

## Output format

Each evaluator returns only:
- **Verdict:** PASS / PASS WITH RISKS / REJECT
- **Evidence:** concrete observations
- **Strongest objection**
- **Next experiment/fix**

No evaluator may approve based on aesthetics or novelty.
