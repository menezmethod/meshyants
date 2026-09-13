package evalgate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrInvalidReport = errors.New("evalgate: invalid report")
	ErrBlocked       = errors.New("evalgate: work blocked")
	ErrTooLarge      = errors.New("evalgate: report exceeds size bound")
)

var vacuousWhole = regexp.MustCompile(`(?i)^(lgtm|looks good\.?|seems fine\.?|ship it\.?|elegant\.?|novel\.?|clean( code| architecture)?\.?)$`)

func LoadReport(path string) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() > int64(MaxReportBytes) {
		return nil, ErrTooLarge
	}
	return DecodeReport(io.LimitReader(f, int64(MaxReportBytes)+1))
}

func DecodeReport(r io.Reader) (*Report, error) {
	limited := io.LimitReader(r, MaxReportBytes+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(b) > MaxReportBytes {
		return nil, ErrTooLarge
	}
	var report Report
	if err := json.Unmarshal(b, &report); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	return &report, nil
}

func Validate(report *Report) error {
	if report == nil {
		return fmt.Errorf("%w: nil report", ErrInvalidReport)
	}
	if report.SchemaVersion != SchemaVersion {
		return fmt.Errorf("%w: schema_version must be %d", ErrInvalidReport, SchemaVersion)
	}
	if strings.TrimSpace(report.TaskID) == "" {
		return fmt.Errorf("%w: task_id required", ErrInvalidReport)
	}
	if strings.TrimSpace(report.Hypothesis) == "" {
		return fmt.Errorf("%w: hypothesis required", ErrInvalidReport)
	}
	if strings.TrimSpace(report.FalsificationCondition) == "" {
		return fmt.Errorf("%w: falsification_condition required", ErrInvalidReport)
	}
	if report.Round < 1 {
		return fmt.Errorf("%w: round must be >= 1", ErrInvalidReport)
	}
	if !report.CompositeVerdict.Valid() {
		return fmt.Errorf("%w: composite_verdict %q", ErrInvalidReport, report.CompositeVerdict)
	}
	if err := report.Scorecard.inRange(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidReport, err)
	}
	if report.Scorecard.HiddenZero() && report.CompositeVerdict == VerdictPass {
		return fmt.Errorf("%w: hidden zero dimension behind total %d; cannot PASS", ErrInvalidReport, report.Scorecard.Total())
	}
	if report.Scorecard.SimplerBaselineAdvantage >= 3 && strings.TrimSpace(report.BaselineResults) == "" {
		return fmt.Errorf("%w: simpler_baseline_advantage >= 3 requires baseline_results", ErrInvalidReport)
	}

	seen := map[EvaluatorID]int{}
	for i, ev := range report.Evaluators {
		if err := validateEvaluator(report, i, ev); err != nil {
			return err
		}
		seen[ev.ID]++
	}
	for _, id := range RequiredEvaluators {
		n := seen[id]
		if n == 0 {
			return fmt.Errorf("%w: missing evaluator %s", ErrInvalidReport, id)
		}
		if n > 1 {
			return fmt.Errorf("%w: duplicate evaluator %s", ErrInvalidReport, id)
		}
	}
	if extra := extraEvaluators(seen); extra != "" {
		return fmt.Errorf("%w: unknown evaluator %s", ErrInvalidReport, extra)
	}

	derived := deriveVerdict(report)
	if derived != report.CompositeVerdict {
		return fmt.Errorf("%w: composite_verdict %s disagrees with derived %s", ErrInvalidReport, report.CompositeVerdict, derived)
	}

	for i, f := range report.Findings {
		if err := validateFinding(fmt.Sprintf("findings[%d]", i), f); err != nil {
			return err
		}
	}
	return nil
}

func extraEvaluators(seen map[EvaluatorID]int) string {
	allowed := map[EvaluatorID]struct{}{}
	for _, id := range RequiredEvaluators {
		allowed[id] = struct{}{}
	}
	for id := range seen {
		if _, ok := allowed[id]; !ok {
			return string(id)
		}
	}
	return ""
}

func validateEvaluator(report *Report, i int, ev EvaluatorResult) error {
	prefix := fmt.Sprintf("evaluators[%d]", i)
	if !knownEvaluator(ev.ID) {
		return fmt.Errorf("%w: %s unknown id %s", ErrInvalidReport, prefix, ev.ID)
	}
	if ev.NotApplicable {
		if strings.TrimSpace(ev.NotApplicableReason) == "" {
			return fmt.Errorf("%w: %s not_applicable requires a reason", ErrInvalidReport, prefix)
		}
		if !ev.Verdict.Valid() {
			return fmt.Errorf("%w: %s verdict %q", ErrInvalidReport, prefix, ev.Verdict)
		}
		return nil
	}
	if !ev.Verdict.Valid() {
		return fmt.Errorf("%w: %s verdict %q", ErrInvalidReport, prefix, ev.Verdict)
	}
	if strings.TrimSpace(ev.StrongestSimplerExplanation) == "" {
		return fmt.Errorf("%w: %s strongest_simpler_explanation required", ErrInvalidReport, prefix)
	}
	if strings.TrimSpace(ev.StrongestObjection) == "" {
		return fmt.Errorf("%w: %s strongest_objection required", ErrInvalidReport, prefix)
	}
	if strings.TrimSpace(ev.TargetChallenge) == "" {
		return fmt.Errorf("%w: %s target_challenge required", ErrInvalidReport, prefix)
	}
	if strings.TrimSpace(ev.NextExperiment) == "" {
		return fmt.Errorf("%w: %s next_experiment required", ErrInvalidReport, prefix)
	}
	if ev.RepeatedBlocker && strings.TrimSpace(ev.RepeatedBlockerWhat) == "" {
		return fmt.Errorf("%w: %s repeated_blocker_what required", ErrInvalidReport, prefix)
	}
	if allEvidenceVacuous(ev.Evidence) {
		return fmt.Errorf("%w: %s evidence is vacuous/aesthetic-only; cannot approve on novelty or elegance", ErrInvalidReport, prefix)
	}
	if err := validateSpecialized(prefix, ev); err != nil {
		return err
	}
	if ev.ID == SchedulerCritic && (ev.SimplerSystemOutcome == "wins" || ev.SimplerSystemOutcome == "ties") {
		if report.Scorecard.SimplerBaselineAdvantage >= 3 {
			return fmt.Errorf("%w: %s simpler system %s but simpler_baseline_advantage=%d", ErrInvalidReport, prefix, ev.SimplerSystemOutcome, report.Scorecard.SimplerBaselineAdvantage)
		}
	}
	for j, f := range ev.Findings {
		if err := validateFinding(fmt.Sprintf("%s.findings[%d]", prefix, j), f); err != nil {
			return err
		}
	}
	return nil
}

func knownEvaluator(id EvaluatorID) bool {
	for _, want := range RequiredEvaluators {
		if id == want {
			return true
		}
	}
	return false
}

func validateSpecialized(prefix string, ev EvaluatorResult) error {
	switch ev.ID {
	case ScientificSkeptic:
		if strings.TrimSpace(ev.WhatWouldReverseConclusion) == "" {
			return fmt.Errorf("%w: %s what_would_reverse_conclusion required", ErrInvalidReport, prefix)
		}
	case SchedulerCritic:
		if _, ok := AllowedSimplerSystems[ev.SimplerSystem]; !ok {
			return fmt.Errorf("%w: %s simpler_system %q not in allowed set", ErrInvalidReport, prefix, ev.SimplerSystem)
		}
		if ev.SimplerSystem == "other" && len(strings.TrimSpace(ev.StrongestSimplerExplanation)) < 40 {
			return fmt.Errorf("%w: %s simpler_system=other needs a longer explanation", ErrInvalidReport, prefix)
		}
		if _, ok := AllowedSimplerOutcomes[ev.SimplerSystemOutcome]; !ok {
			return fmt.Errorf("%w: %s simpler_system_outcome %q", ErrInvalidReport, prefix, ev.SimplerSystemOutcome)
		}
	case PerformanceCritic:
		if len(ev.Measurements) == 0 {
			return fmt.Errorf("%w: %s measurements required (use not_applicable if no runtime change)", ErrInvalidReport, prefix)
		}
	case FailureCritic:
		if len(ev.PerturbationsConsidered) == 0 {
			return fmt.Errorf("%w: %s perturbations_considered required", ErrInvalidReport, prefix)
		}
	case SlopEvaluator:
		if strings.TrimSpace(ev.InvariantNamed) == "" {
			return fmt.Errorf("%w: %s invariant_named required", ErrInvalidReport, prefix)
		}
		if strings.TrimSpace(ev.DeletionCandidate) == "" {
			return fmt.Errorf("%w: %s deletion_candidate required", ErrInvalidReport, prefix)
		}
	}
	return nil
}

func validateFinding(prefix string, f Finding) error {
	switch f.Severity {
	case "reject", "risk", "observation":
	default:
		return fmt.Errorf("%w: %s severity %q", ErrInvalidReport, prefix, f.Severity)
	}
	if strings.TrimSpace(f.Summary) == "" {
		return fmt.Errorf("%w: %s summary required", ErrInvalidReport, prefix)
	}
	if strings.TrimSpace(f.Evidence) == "" {
		return fmt.Errorf("%w: %s evidence required", ErrInvalidReport, prefix)
	}
	if f.Severity == "reject" || f.Severity == "risk" {
		if f.ProposedTask == nil {
			return fmt.Errorf("%w: %s %s finding must propose a task (do not bury negatives)", ErrInvalidReport, prefix, f.Severity)
		}
		if err := validateDraft(prefix+".proposed_task", f.ProposedTask); err != nil {
			return err
		}
	}
	return nil
}

func validateDraft(prefix string, d *TaskDraft) error {
	if strings.TrimSpace(d.Title) == "" {
		return fmt.Errorf("%w: %s title required", ErrInvalidReport, prefix)
	}
	if strings.TrimSpace(d.Hypothesis) == "" {
		return fmt.Errorf("%w: %s hypothesis required", ErrInvalidReport, prefix)
	}
	if strings.TrimSpace(d.FalsificationCondition) == "" {
		return fmt.Errorf("%w: %s falsification_condition required", ErrInvalidReport, prefix)
	}
	return nil
}

func allEvidenceVacuous(evidence []string) bool {
	if len(evidence) == 0 {
		return true
	}
	for _, e := range evidence {
		if !vacuousEvidence(e) {
			return false
		}
	}
	return true
}

var fileExt = regexp.MustCompile(`(?i)[A-Za-z0-9._-]+\.(go|json|md|yml|yaml|txt|proto)`)

func hasConcreteAnchor(s string) bool {
	if strings.ContainsAny(s, "/\\") {
		return true
	}
	if strings.Contains(s, "://") {
		return true
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "testdata") {
		return true
	}
	if strings.Contains(s, "MESH-") {
		return true
	}
	if fileExt.MatchString(s) {
		return true
	}
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func vacuousEvidence(e string) bool {
	s := strings.TrimSpace(e)
	if s == "" {
		return true
	}
	if vacuousWhole.MatchString(s) {
		return true
	}
	return !hasConcreteAnchor(s)
}

func deriveVerdict(report *Report) Verdict {
	verdict := VerdictPass
	for _, ev := range report.Evaluators {
		if ev.NotApplicable {
			continue
		}
		switch ev.Verdict {
		case VerdictReject:
			return VerdictReject
		case VerdictPassWithRisks:
			verdict = VerdictPassWithRisks
		}
	}
	for _, f := range allFindings(report) {
		if f.Severity == "reject" {
			return VerdictReject
		}
		if f.Severity == "risk" && verdict == VerdictPass {
			verdict = VerdictPassWithRisks
		}
	}
	return verdict
}

func allFindings(report *Report) []Finding {
	out := append([]Finding{}, report.Findings...)
	for _, ev := range report.Evaluators {
		out = append(out, ev.Findings...)
	}
	return out
}

// Decide validates the report and returns whether the work may advance.
// REJECT (evaluator, composite, or finding) blocks. Risk/reject findings spawn drafts.
func Decide(report *Report) (*Decision, error) {
	if err := Validate(report); err != nil {
		return nil, err
	}
	d := &Decision{
		Verdict:   report.CompositeVerdict,
		Scorecard: report.Scorecard,
		Total:     report.Scorecard.Total(),
		Spawned:   Spawn(report),
	}
	if report.CompositeVerdict == VerdictReject {
		d.Advance = false
		d.Reason = "composite REJECT"
		d.Blockers = append(d.Blockers, "composite_verdict=REJECT")
		return d, nil
	}
	d.Advance = true
	if report.CompositeVerdict == VerdictPassWithRisks {
		d.Reason = "PASS WITH RISKS; spawned task drafts must not be buried"
	} else {
		d.Reason = "PASS"
	}
	return d, nil
}

// Spawn extracts task drafts from reject/risk findings. Observation findings do not spawn.
func Spawn(report *Report) []TaskDraft {
	if report == nil {
		return nil
	}
	var out []TaskDraft
	n := 0
	for _, f := range allFindings(report) {
		if f.Severity != "reject" && f.Severity != "risk" {
			continue
		}
		if f.ProposedTask == nil {
			continue
		}
		n++
		draft := *f.ProposedTask
		if draft.SourceTaskID == "" {
			draft.SourceTaskID = report.TaskID
		}
		if draft.SourceFinding == "" {
			draft.SourceFinding = f.Summary
		}
		if draft.SuggestedID == "" {
			draft.SuggestedID = fmt.Sprintf("%s-F%d", report.TaskID, n)
		}
		out = append(out, draft)
	}
	return out
}

// WriteSpawned persists drafts so negatives are not buried in chat.
func WriteSpawned(path string, drafts []TaskDraft) error {
	if drafts == nil {
		drafts = []TaskDraft{}
	}
	b, err := json.MarshalIndent(drafts, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

// TaskMayBecomeDone is the status gate: done requires a valid advancing report.
func TaskMayBecomeDone(status string, report *Report) error {
	if status != "done" {
		return nil
	}
	d, err := Decide(report)
	if err != nil {
		return err
	}
	if !d.Advance {
		return fmt.Errorf("%w: %s", ErrBlocked, d.Reason)
	}
	return nil
}

// CheckTasks is the research-loop done hook: every task with status=done
// must have resultsDir/<id>/eval-report.json that Decide allows to advance.
// It only reads tasks.json; it never rewrites the shared queue.
func CheckTasks(tasksPath, resultsDir string) error {
	b, err := os.ReadFile(tasksPath)
	if err != nil {
		return err
	}
	if len(b) > MaxReportBytes {
		return ErrTooLarge
	}
	var doc struct {
		Tasks []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return fmt.Errorf("%w: tasks.json: %v", ErrInvalidReport, err)
	}
	var blockers []string
	for _, t := range doc.Tasks {
		if t.Status != "done" {
			continue
		}
		reportPath := filepath.Join(resultsDir, t.ID, "eval-report.json")
		report, err := LoadReport(reportPath)
		if err != nil {
			blockers = append(blockers, fmt.Sprintf("%s: missing or unreadable %s (%v)", t.ID, reportPath, err))
			continue
		}
		if err := TaskMayBecomeDone("done", report); err != nil {
			blockers = append(blockers, fmt.Sprintf("%s: %v", t.ID, err))
		}
	}
	if len(blockers) > 0 {
		return fmt.Errorf("%w: %s", ErrBlocked, strings.Join(blockers, "; "))
	}
	return nil
}
