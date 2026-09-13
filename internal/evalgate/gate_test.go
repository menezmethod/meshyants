package evalgate_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meshyants/meshyants/v1/internal/evalgate"
	"github.com/stretchr/testify/require"
)

func validPass() *evalgate.Report {
	return &evalgate.Report{
		SchemaVersion:          evalgate.SchemaVersion,
		TaskID:                 "MESH-108",
		Hypothesis:             "A machine-checked gauntlet rejects rubber-stamp reviews and spawns tasks from negatives.",
		FalsificationCondition: "A vacuous PASS advances, or a REJECT fails to block done, or a negative finding produces no draft.",
		Round:                  1,
		CompositeVerdict:       evalgate.VerdictPass,
		Scorecard: evalgate.Scorecard{
			ScientificValidity:       3,
			SimplerBaselineAdvantage: 3,
			PerformanceCost:          3,
			ResilienceScalability:    2,
			CodeExperimentQuality:    3,
		},
		BaselineResults:       "checklist_only accepted the rubber-stamp fixture; evalgate rejected it",
		CostLatencyModelCalls: "gate decide <1ms; 0 model calls",
		Evaluators: []evalgate.EvaluatorResult{
			{
				ID:                          evalgate.ScientificSkeptic,
				Verdict:                     evalgate.VerdictPass,
				Evidence:                    []string{"testdata/rubber_stamp.json is rejected by Validate; testdata/valid_reject.json blocks TaskMayBecomeDone"},
				StrongestObjection:          "Structural checks can be satisfied with fluent but empty prose if someone tries.",
				StrongestSimplerExplanation: "A careful human reading EVALUATORS.md would also reject rubber stamps.",
				TargetChallenge:             "The hypothesis may overclaim: schema enforcement is not independent scientific judgment.",
				NextExperiment:              "Feed a fluent-but-empty report that fills every required field and see if it still advances.",
				WhatWouldReverseConclusion:  "A rubber-stamp or missing-simpler-explanation report that Decide allows to advance.",
			},
			{
				ID:                          evalgate.SchedulerCritic,
				Verdict:                     evalgate.VerdictPass,
				Evidence:                    []string{"docs/tasks/eval-report.schema.json accepts testdata/rubber_stamp.json; evalgate Validate rejects it."},
				StrongestObjection:          "A JSON Schema plus a 50-line validator could encode the same reject/spawn rules.",
				StrongestSimplerExplanation: "Required fields on round-N.json plus a schema file, without a Go package.",
				TargetChallenge:             "no — the target is a process gate, not a colony benchmark",
				NextExperiment:              "Compare loc and false-negative rate against a schema-only checker.",
				SimplerSystem:               "checklist_or_schema",
				SimplerSystemOutcome:        "loses",
			},
			{
				ID:                          evalgate.PerformanceCritic,
				Verdict:                     evalgate.VerdictPass,
				Evidence:                    []string{"TestGate_DecideLatency in internal/evalgate/gate_test.go measures 200 Decide calls under 200ms."},
				StrongestObjection:          "No production CI hook yet, so the gate can be skipped by not invoking it.",
				StrongestSimplerExplanation: "Document the checklist and do not spend compile time on a gate binary.",
				TargetChallenge:             "no",
				NextExperiment:              "Add an optional CI step that validates docs/results/**/eval-report.json when present.",
				Measurements:                map[string]string{"decide_ns_budget": "5000000", "model_calls": "0"},
			},
			{
				ID:                          evalgate.FailureCritic,
				Verdict:                     evalgate.VerdictPass,
				Evidence:                    []string{"internal/evalgate/gate.go DecodeReport rejects oversized payloads; TestGate_MalformedAndHugeInputs covers nil and malformed JSON."},
				StrongestObjection:          "There is no lease/fencing surface here; this is a file-shaped report, not a worker protocol.",
				StrongestSimplerExplanation: "os.ReadFile plus json.Unmarshal without size bounds would be the naive parser.",
				TargetChallenge:             "no",
				NextExperiment:              "Replay a truncated and a duplicate-evaluator report.",
				PerturbationsConsidered:     []string{"oversized_json", "malformed_json", "nil_report", "duplicate_evaluator", "stale_schema_version"},
			},
			{
				ID:                          evalgate.SlopEvaluator,
				Verdict:                     evalgate.VerdictPass,
				Evidence:                    []string{"internal/evalgate/gate_test.go names scenario tests (rubber stamp, buried finding, fluent aesthetic) rather than field-by-field mirrors."},
				StrongestObjection:          "cmd/evalgate could be deleted; go test is enough to exercise the package.",
				StrongestSimplerExplanation: "Keep only the package and schema; drop the CLI until an agent actually invokes it.",
				TargetChallenge:             "no",
				NextExperiment:              "Delete the CLI if the next round shows no caller.",
				InvariantNamed:              "REJECT or reject-severity findings must not allow status=done; risk/reject findings must propose a task.",
				DeletionCandidate:           "cmd/evalgate is convenience only; the package is the gate.",
			},
		},
	}
}

func TestGate_RejectsRubberStampPass(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Evaluators[0].Evidence = []string{"looks good", "elegant", "novel"}
	r.Evaluators[0].StrongestSimplerExplanation = "none"
	err := evalgate.Validate(r)
	require.Error(t, err)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "vacuous")
}

func TestGate_RejectsMissingSimplerExplanation(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Evaluators[1].StrongestSimplerExplanation = "   "
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "strongest_simpler_explanation")
}

func TestGate_RejectBlocksDone(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.CompositeVerdict = evalgate.VerdictReject
	r.Evaluators[0].Verdict = evalgate.VerdictReject
	r.Evaluators[0].Findings = []evalgate.Finding{{
		Severity: "reject",
		Summary:  "Claimed advantage has no baseline run",
		Evidence: "baseline_results names no artifact path or metric",
		ProposedTask: &evalgate.TaskDraft{
			Title:                  "Add a non-handicapped baseline comparison",
			Workstream:             "quality",
			Type:                   "experiment",
			Hypothesis:             "The claimed gate advantage disappears against a schema-only checker.",
			FalsificationCondition: "Schema-only checker rejects the same rubber stamps and spawns the same drafts.",
		},
	}}
	require.NoError(t, evalgate.Validate(r))
	d, err := evalgate.Decide(r)
	require.NoError(t, err)
	require.False(t, d.Advance)
	require.Equal(t, evalgate.VerdictReject, d.Verdict)
	require.ErrorIs(t, evalgate.TaskMayBecomeDone("done", r), evalgate.ErrBlocked)
	require.NoError(t, evalgate.TaskMayBecomeDone("in_progress", r))
}

func TestGate_NegativeFindingSpawnsTaskDraft(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.CompositeVerdict = evalgate.VerdictPassWithRisks
	r.Evaluators[4].Verdict = evalgate.VerdictPassWithRisks
	r.Evaluators[4].Findings = []evalgate.Finding{{
		Severity: "risk",
		Summary:  "CLI is unused convenience",
		Evidence: "no caller of cmd/evalgate in existing scripts or CI",
		ProposedTask: &evalgate.TaskDraft{
			Title:                  "Prove or delete evalgate CLI",
			Workstream:             "quality",
			Type:                   "implementation",
			Hypothesis:             "Agents will invoke cmd/evalgate on material reports.",
			FalsificationCondition: "After two loops no report is passed through the CLI.",
		},
	}}
	d, err := evalgate.Decide(r)
	require.NoError(t, err)
	require.True(t, d.Advance)
	require.Len(t, d.Spawned, 1)
	require.Equal(t, "MESH-108-F1", d.Spawned[0].SuggestedID)
	require.Equal(t, "MESH-108", d.Spawned[0].SourceTaskID)
	require.NotEmpty(t, d.Spawned[0].Hypothesis)
	require.NotEmpty(t, d.Spawned[0].FalsificationCondition)

	dir := t.TempDir()
	path := filepath.Join(dir, "spawned-tasks.json")
	require.NoError(t, evalgate.WriteSpawned(path, d.Spawned))
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var loaded []evalgate.TaskDraft
	require.NoError(t, json.Unmarshal(b, &loaded))
	require.Equal(t, d.Spawned, loaded)
}

func TestGate_BuriedNegativeFindingRejected(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Findings = []evalgate.Finding{{
		Severity: "reject",
		Summary:  "Benchmark leakage",
		Evidence: "test fixture is also the claimed production benchmark",
	}}
	r.CompositeVerdict = evalgate.VerdictReject
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "must propose a task")
}

func TestGate_AdvantageWithoutBaselineRejected(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.BaselineResults = ""
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "baseline_results")
}

func TestGate_InconsistentCompositeRejected(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Evaluators[2].Verdict = evalgate.VerdictReject
	// composite left as PASS
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "disagrees")
}

func TestGate_SchedulerWinInconsistentWithAdvantageScore(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Evaluators[1].SimplerSystemOutcome = "ties"
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "simpler_baseline_advantage")
}

func TestGate_HiddenZeroCannotPass(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Scorecard.ResilienceScalability = 0
	r.Scorecard.ScientificValidity = 5
	r.Scorecard.SimplerBaselineAdvantage = 5
	r.Scorecard.PerformanceCost = 5
	r.Scorecard.CodeExperimentQuality = 5
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "hidden zero")
}

func TestGate_SchedulerMustNameAllowedSimplerSystem(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Evaluators[1].SimplerSystem = "magic_swarm"
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "simpler_system")
}

func TestGate_ValidPassAdvances(t *testing.T) {
	t.Parallel()
	r := validPass()
	d, err := evalgate.Decide(r)
	require.NoError(t, err)
	require.True(t, d.Advance)
	require.Equal(t, 14, d.Total)
	require.NoError(t, evalgate.TaskMayBecomeDone("done", r))
}

func TestGate_NotApplicableRequiresReason(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Evaluators[2] = evalgate.EvaluatorResult{
		ID:            evalgate.PerformanceCritic,
		Verdict:       evalgate.VerdictPass,
		NotApplicable: true,
	}
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "not_applicable")

	r.Evaluators[2].NotApplicableReason = "docs-only prompt packets; no runtime path changed besides tests"
	require.NoError(t, evalgate.Validate(r))
}

func TestGate_RejectsFluentAestheticEvidence(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Evaluators[0].Evidence = []string{"The architecture is elegant and clearly better than a queue."}
	err := evalgate.Validate(r)
	require.ErrorIs(t, err, evalgate.ErrInvalidReport)
	require.Contains(t, err.Error(), "vacuous")
}

// schemaOnlyAccepts is the non-handicapped simpler baseline: required-field
// presence as docs/tasks/eval-report.schema.json encodes it (id+verdict on
// evaluators, no vacuity rule).
func schemaOnlyAccepts(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return false
	}
	for _, k := range []string{
		"schema_version", "task_id", "hypothesis", "falsification_condition",
		"round", "composite_verdict", "scorecard", "evaluators",
		"baseline_results", "cost_latency_model_calls",
	} {
		if _, ok := raw[k]; !ok {
			return false
		}
	}
	evs, ok := raw["evaluators"].([]any)
	if !ok || len(evs) < 5 {
		return false
	}
	for _, e := range evs {
		m, ok := e.(map[string]any)
		if !ok {
			return false
		}
		if m["id"] == nil || m["verdict"] == nil {
			return false
		}
	}
	return true
}

func TestGate_SchemaOnlyBaselineLosesOnVacuity(t *testing.T) {
	t.Parallel()
	require.True(t, schemaOnlyAccepts("testdata/valid_pass.json"))
	require.True(t, schemaOnlyAccepts("testdata/valid_reject.json"))
	require.True(t, schemaOnlyAccepts("testdata/rubber_stamp.json"), "schema-only must accept the rubber stamp (no vacuity rule)")
	require.True(t, schemaOnlyAccepts("testdata/fluent_stamp.json"), "schema-only must accept fluent aesthetic evidence")

	stamp, err := evalgate.LoadReport("testdata/fluent_stamp.json")
	require.NoError(t, err)
	require.Error(t, evalgate.Validate(stamp), "evalgate must reject fluent aesthetic evidence that schema-only accepts")
}

func TestGate_LoadFixtures(t *testing.T) {
	t.Parallel()
	pass, err := evalgate.LoadReport("testdata/valid_pass.json")
	require.NoError(t, err)
	d, err := evalgate.Decide(pass)
	require.NoError(t, err)
	require.True(t, d.Advance)

	rej, err := evalgate.LoadReport("testdata/valid_reject.json")
	require.NoError(t, err)
	d, err = evalgate.Decide(rej)
	require.NoError(t, err)
	require.False(t, d.Advance)
	require.NotEmpty(t, d.Spawned)

	stamp, err := evalgate.LoadReport("testdata/rubber_stamp.json")
	require.NoError(t, err)
	require.Error(t, evalgate.Validate(stamp))
}

func TestGate_MalformedAndHugeInputs(t *testing.T) {
	t.Parallel()
	_, err := evalgate.DecodeReport(strings.NewReader("{not json"))
	require.Error(t, err)

	huge := bytes.Repeat([]byte("a"), evalgate.MaxReportBytes+8)
	_, err = evalgate.DecodeReport(bytes.NewReader(huge))
	require.ErrorIs(t, err, evalgate.ErrTooLarge)

	require.Error(t, evalgate.Validate(nil))
	_, err = evalgate.Decide(nil)
	require.Error(t, err)
	require.Nil(t, evalgate.Spawn(nil))
}

func TestGate_StaleSchemaAndDuplicates(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.SchemaVersion = 0
	require.Error(t, evalgate.Validate(r))

	r = validPass()
	r.Evaluators = append(r.Evaluators, r.Evaluators[0])
	err := evalgate.Validate(r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")

	r = validPass()
	r.Evaluators[0].ID = "vibes_critic"
	err = evalgate.Validate(r)
	require.Error(t, err)
}

func TestGate_ScoreOutOfRange(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Scorecard.ScientificValidity = 6
	require.Error(t, evalgate.Validate(r))
}

func TestGate_DecideLatency(t *testing.T) {
	t.Parallel()
	r := validPass()
	start := time.Now()
	for i := 0; i < 200; i++ {
		_, err := evalgate.Decide(r)
		require.NoError(t, err)
	}
	elapsed := time.Since(start)
	require.Less(t, elapsed, 200*time.Millisecond, "200 Decide calls should stay cheap")
}

func TestLoadReport_BoundedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.json")
	require.NoError(t, os.WriteFile(path, bytes.Repeat([]byte("x"), evalgate.MaxReportBytes+8), 0o644))
	_, err := evalgate.LoadReport(path)
	require.ErrorIs(t, err, evalgate.ErrTooLarge)
}

func TestCheckTasks_DoneRequiresAdvancingReport(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	results := filepath.Join(dir, "results")
	require.NoError(t, os.MkdirAll(filepath.Join(results, "MESH-X"), 0o755))

	writeTasks := func(status string) string {
		p := filepath.Join(dir, "tasks-"+status+".json")
		body := fmt.Sprintf(`{"tasks":[{"id":"MESH-X","status":%q}]}`, status)
		require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
		return p
	}

	require.NoError(t, evalgate.CheckTasks(writeTasks("in_progress"), results), "in_progress needs no report")

	err := evalgate.CheckTasks(writeTasks("done"), results)
	require.ErrorIs(t, err, evalgate.ErrBlocked, "done with missing report must block")

	pass, err := os.ReadFile("testdata/valid_pass.json")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(results, "MESH-X", "eval-report.json"), pass, 0o644))
	require.NoError(t, evalgate.CheckTasks(writeTasks("done"), results))

	rej, err := os.ReadFile("testdata/valid_reject.json")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(results, "MESH-X", "eval-report.json"), rej, 0o644))
	err = evalgate.CheckTasks(writeTasks("done"), results)
	require.ErrorIs(t, err, evalgate.ErrBlocked)
}

func TestCheckTasks_RealQueueHasNoDoneWithoutReport(t *testing.T) {
	t.Parallel()
	err := evalgate.CheckTasks("../../docs/tasks/tasks.json", "../../docs/results")
	require.NoError(t, err, "current queue has no done tasks; hook must be a no-op")
}

func TestGate_ObservationFindingDoesNotSpawn(t *testing.T) {
	t.Parallel()
	r := validPass()
	r.Findings = []evalgate.Finding{{
		Severity: "observation",
		Summary:  "No CI hook yet",
		Evidence: "ci.yml does not invoke evalgate",
	}}
	d, err := evalgate.Decide(r)
	require.NoError(t, err)
	require.Empty(t, d.Spawned)
}
