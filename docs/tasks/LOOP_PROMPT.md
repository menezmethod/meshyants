# MeshyAnts Autonomous Research/Build Loop Prompt

You are working on **MeshyAnts**, an experimental substrate for collective machine intelligence.

Your goal is **not to build another shiny agent framework**. Your goal is to discover whether biologically inspired decentralized coordination produces measurable advantages over simpler orchestration, and to publish honest evidence either way.

## Read first

1. `README.md`
2. `docs/revival/RESEARCH.md`
3. `docs/revival/ARCHITECTURE.md`
4. `docs/revival/EXPERIMENTS.md`
5. `docs/revival/MENTAT.md`
6. `docs/tasks/tasks.json`
7. `docs/tasks/EVALUATORS.md`

## Execute one loop

1. **Select:** choose the highest-priority unblocked task that advances an experiment.
2. **Frame:** state the claim, strongest simpler alternative, missing evidence, and falsification condition.
3. **Act:** implement/research the smallest useful slice. Reuse existing v1 substrate before adding infrastructure.
4. **Measure:** run tests plus the relevant benchmark/performance/failure experiments. Save machine-readable evidence where possible.
5. **Attack:** run independent evaluator passes from `EVALUATORS.md`. The scheduler critic must actively try to show that a simpler system explains the result.
6. **Decide:** keep, revise, or reject the approach based on evidence—not sunk cost.
7. **Record:** update docs/results and `tasks.json`. New discoveries may create tasks. Unsupported ideas stay hypotheses.
8. **Repeat:** continue only while there is a clear unblocked next task and the previous step has evidence.

## Research loop

When a design fork needs outside evidence, or a meaningful new NeuroAI/connectomics result appears:
- search current primary literature first;
- use FlyWire/connectomics, Princeton NeuroAI, Allen Institute, Neuromatch and relevant complex-systems work;
- podcasts/transcripts such as Brain Inspired are useful discovery sources, but trace important claims back to primary work;
- translate biology into **mechanism → software hypothesis → experiment**;
- include a "why this analogy may fail" section.

Do not copy biology because it sounds elegant.

## North-star outcome

Work toward a reproducible run where MeshyAnts:
- discovers or adapts a useful workflow/path that was not explicitly scripted;
- survives a meaningful perturbation;
- and beats a credible baseline on quality, cost, latency, adaptation, or resilience.

The WOW is **measured unexpected capability**, not a visualization of many agents talking.

## Community

Design for outsiders to eventually contribute reproducible worker phenotypes/skills ("ants"), but do not build a marketplace until the core experiment earns it. Any community ant must have a versioned manifest, scoped permissions, provenance, and benchmark evidence.

## Stop conditions

Stop and create a finding/task instead of coding when:
- the experiment cannot distinguish MeshyAnts from a simpler scheduler;
- metrics are missing;
- the task requires inventing unapproved architecture;
- evaluators reject the result;
- the biological analogy has no measurable software prediction.

Negative results are progress.
