// Package evalgate is the MESH-108 skeptic/evaluator gauntlet: a machine-checked
// report contract that can reject work, requires a strongest simpler
// explanation, and turns negative findings into task drafts.
package evalgate

import "fmt"

const SchemaVersion = 1

// MaxReportBytes is a malformed-input bound (failure critic: do not panic on huge JSON).
const MaxReportBytes = 256 << 10

type Verdict string

const (
	VerdictPass          Verdict = "PASS"
	VerdictPassWithRisks Verdict = "PASS WITH RISKS"
	VerdictReject        Verdict = "REJECT"
)

func (v Verdict) Valid() bool {
	switch v {
	case VerdictPass, VerdictPassWithRisks, VerdictReject:
		return true
	default:
		return false
	}
}

type EvaluatorID string

const (
	ScientificSkeptic EvaluatorID = "scientific_skeptic"
	SchedulerCritic   EvaluatorID = "scheduler_critic"
	PerformanceCritic EvaluatorID = "performance_critic"
	FailureCritic     EvaluatorID = "failure_critic"
	SlopEvaluator     EvaluatorID = "slop_evaluator"
)

// RequiredEvaluators is the full gauntlet. An evaluator may be marked
// not_applicable with a reason; omission is not allowed.
var RequiredEvaluators = []EvaluatorID{
	ScientificSkeptic,
	SchedulerCritic,
	PerformanceCritic,
	FailureCritic,
	SlopEvaluator,
}

// AllowedSimplerSystems are the scheduler critic's named alternatives
// from docs/tasks/EVALUATORS.md plus the current unenforced-checklist baseline.
var AllowedSimplerSystems = map[string]struct{}{
	"priority_queue":      {},
	"central_router":      {},
	"deterministic_dag":   {},
	"manager_worker":      {},
	"single_worker":       {},
	"checklist_or_schema": {},
	"other":               {},
}

var AllowedSimplerOutcomes = map[string]struct{}{
	"wins":    {},
	"ties":    {},
	"loses":   {},
	"unknown": {},
}

// Scorecard is the stable 0–5 / 5-dimension card from EVALUATORS.md.
type Scorecard struct {
	ScientificValidity       int `json:"scientific_validity"`
	SimplerBaselineAdvantage int `json:"simpler_baseline_advantage"`
	PerformanceCost          int `json:"performance_cost"`
	ResilienceScalability    int `json:"resilience_scalability"`
	CodeExperimentQuality    int `json:"code_experiment_quality"`
}

func (s Scorecard) Values() []int {
	return []int{
		s.ScientificValidity,
		s.SimplerBaselineAdvantage,
		s.PerformanceCost,
		s.ResilienceScalability,
		s.CodeExperimentQuality,
	}
}

func (s Scorecard) Total() int {
	t := 0
	for _, v := range s.Values() {
		t += v
	}
	return t
}

func (s Scorecard) HiddenZero() bool {
	if s.Total() < 15 {
		return false
	}
	for _, v := range s.Values() {
		if v == 0 {
			return true
		}
	}
	return false
}

func (s Scorecard) inRange() error {
	for i, v := range s.Values() {
		if v < 0 || v > 5 {
			return fmt.Errorf("scorecard dimension %d out of range 0-5: %d", i, v)
		}
	}
	return nil
}

// Finding is a negative or risk observation. Reject/risk findings must propose a task.
type Finding struct {
	Severity     string     `json:"severity"` // reject | risk | observation
	Summary      string     `json:"summary"`
	Evidence     string     `json:"evidence"`
	ProposedTask *TaskDraft `json:"proposed_task,omitempty"`
}

// TaskDraft is a finding-spawned task. It is written to a results artifact,
// not silently dropped, and is not auto-inserted into the shared tasks.json
// queue (avoids colliding with parallel agents).
type TaskDraft struct {
	SuggestedID            string   `json:"suggested_id,omitempty"`
	Title                  string   `json:"title"`
	Workstream             string   `json:"workstream"`
	Type                   string   `json:"type"`
	Priority               int      `json:"priority,omitempty"`
	Hypothesis             string   `json:"hypothesis"`
	FalsificationCondition string   `json:"falsification_condition"`
	Deliverables           []string `json:"deliverables,omitempty"`
	SuccessCriteria        []string `json:"success_criteria,omitempty"`
	SourceTaskID           string   `json:"source_task_id,omitempty"`
	SourceFinding          string   `json:"source_finding,omitempty"`
}

// EvaluatorResult is one critic's output. Builder advocacy must not be copied here.
type EvaluatorResult struct {
	ID                          EvaluatorID       `json:"id"`
	Verdict                     Verdict           `json:"verdict"`
	NotApplicable               bool              `json:"not_applicable,omitempty"`
	NotApplicableReason         string            `json:"not_applicable_reason,omitempty"`
	Evidence                    []string          `json:"evidence"`
	StrongestObjection          string            `json:"strongest_objection"`
	StrongestSimplerExplanation string            `json:"strongest_simpler_explanation"`
	RepeatedBlocker             bool              `json:"repeated_blocker"`
	RepeatedBlockerWhat         string            `json:"repeated_blocker_what,omitempty"`
	TargetChallenge             string            `json:"target_challenge"`
	NextExperiment              string            `json:"next_experiment"`
	Findings                    []Finding         `json:"findings,omitempty"`
	WhatWouldReverseConclusion  string            `json:"what_would_reverse_conclusion,omitempty"`
	SimplerSystem               string            `json:"simpler_system,omitempty"`
	SimplerSystemOutcome        string            `json:"simpler_system_outcome,omitempty"`
	Measurements                map[string]string `json:"measurements,omitempty"`
	PerturbationsConsidered     []string          `json:"perturbations_considered,omitempty"`
	InvariantNamed              string            `json:"invariant_named,omitempty"`
	DeletionCandidate           string            `json:"deletion_candidate,omitempty"`
}

// Report is the machine-readable gauntlet output attached to a material change.
type Report struct {
	SchemaVersion          int               `json:"schema_version"`
	TaskID                 string            `json:"task_id"`
	Hypothesis             string            `json:"hypothesis"`
	FalsificationCondition string            `json:"falsification_condition"`
	Round                  int               `json:"round"`
	PreviousTotal          *int              `json:"previous_total,omitempty"`
	CompositeVerdict       Verdict           `json:"composite_verdict"`
	Scorecard              Scorecard         `json:"scorecard"`
	Evaluators             []EvaluatorResult `json:"evaluators"`
	BaselineResults        string            `json:"baseline_results"`
	CostLatencyModelCalls  string            `json:"cost_latency_model_calls"`
	Findings               []Finding         `json:"findings,omitempty"`
}

// Decision is the gate output: advance or block, plus spawned drafts.
type Decision struct {
	Advance   bool        `json:"advance"`
	Verdict   Verdict     `json:"verdict"`
	Reason    string      `json:"reason"`
	Scorecard Scorecard   `json:"scorecard"`
	Total     int         `json:"total"`
	Spawned   []TaskDraft `json:"spawned,omitempty"`
	Blockers  []string    `json:"blockers,omitempty"`
}
