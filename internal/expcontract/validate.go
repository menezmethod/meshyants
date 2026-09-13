package expcontract

import (
	"fmt"
	"strings"
)

// Validate checks contract invariants. It does not execute the benchmark.
func (s *RunSpec) Validate() error {
	if s == nil {
		return fmt.Errorf("nil run spec")
	}
	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	if s.ContractVersion != Version {
		add("contract_version must be %d", Version)
	}
	if strings.TrimSpace(s.RunID) == "" {
		add("run_id is required")
	}
	switch s.Phase {
	case "A", "B", "C", "D":
	default:
		add("phase must be A, B, C, or D")
	}
	if strings.TrimSpace(s.Hypothesis) == "" {
		add("hypothesis is required")
	}
	if strings.TrimSpace(s.Falsification) == "" {
		add("falsification is required")
	}
	if strings.TrimSpace(s.Workload.ID) == "" {
		add("workload.id is required")
	}
	if !allowedClasses[s.Workload.Class] {
		add("workload.class %q is not allowed", s.Workload.Class)
	}
	if s.Workload.Class == ClassDurableDAG && s.Workload.DecisiveSwarmBenchmark {
		add("durable_dag must not set decisive_swarm_benchmark=true")
	}
	if strings.TrimSpace(s.Workload.TasksRef) == "" {
		add("workload.tasks_ref is required")
	}
	if strings.TrimSpace(s.WorkerPoolRef) == "" {
		add("worker_pool_ref is required")
	}

	seenSys := map[string]bool{}
	for _, sys := range s.Systems {
		if !allowedSystems[sys] {
			add("unknown system %q", sys)
		}
		if seenSys[sys] {
			add("duplicate system %q", sys)
		}
		seenSys[sys] = true
	}
	for _, req := range requiredSystems {
		if !seenSys[req] {
			add("missing required system %s", req)
		}
	}

	if s.Repeats < 3 {
		add("repeats must be >= 3")
	}
	if len(s.Seeds) != s.Repeats {
		add("seeds length (%d) must equal repeats (%d)", len(s.Seeds), s.Repeats)
	}
	seenSeed := map[int]bool{}
	for _, seed := range s.Seeds {
		if seenSeed[seed] {
			add("duplicate seed %d", seed)
		}
		seenSeed[seed] = true
	}
	if strings.TrimSpace(s.RNGAlgorithm) == "" {
		add("rng_algorithm is required")
	}

	if s.AllowMeshyantsCoordinationLLM {
		add("meshyants coordination LLM is a protocol violation (allow_meshyants_coordination_llm must be false)")
	}

	if s.Budget.MaxWallMS < 1 {
		add("budget.max_wall_ms must be >= 1")
	}
	if s.Budget.MaxWorkerWakeups < 1 {
		add("budget.max_worker_wakeups must be >= 1")
	}
	if len(s.Models) == 0 {
		if s.Budget.MaxModelCalls != 0 || s.Budget.MaxInputTokens != 0 || s.Budget.MaxOutputTokens != 0 || s.Budget.MaxCostMicros != 0 {
			add("deterministic runs (empty models) must set token/call/cost caps to 0")
		}
	} else {
		if s.Budget.MaxModelCalls < 1 || s.Budget.MaxInputTokens < 1 || s.Budget.MaxOutputTokens < 1 || s.Budget.MaxCostMicros < 1 {
			add("LLM runs must set model-call, token, and cost caps > 0")
		}
		if strings.TrimSpace(s.Tokenizer) == "" {
			add("tokenizer is required when models is non-empty")
		}
		if len(s.PriceBook) == 0 {
			add("price_book is required when models is non-empty")
		}
		for _, m := range s.Models {
			if strings.TrimSpace(m.ID) == "" || strings.TrimSpace(m.Provider) == "" {
				add("each model needs id and provider")
			}
			if _, ok := s.PriceBook[m.ID]; !ok {
				add("price_book missing model %q", m.ID)
			}
		}
	}

	seenMetric := map[string]bool{}
	for _, m := range s.Metrics {
		if !allowedMetrics[m] {
			add("unknown metric %q", m)
		}
		seenMetric[m] = true
	}
	for _, req := range requiredPrimaryMetrics {
		if !seenMetric[req] {
			add("missing primary metric %s", req)
		}
	}
	if seenSys[SystemMeshyAnts] {
		for _, req := range requiredSecondaryMetrics {
			if !seenMetric[req] {
				add("missing secondary metric %s required for a meshyants claim", req)
			}
		}
	}

	switch s.Comparison.WinRule {
	case WinRulePairedSign80, WinRuleBootstrapCI95:
	default:
		add("comparison.win_rule must be %s or %s", WinRulePairedSign80, WinRuleBootstrapCI95)
	}
	if s.Comparison.Noise.TaskSuccessRate < 0 || s.Comparison.Noise.Quality < 0 || s.Comparison.Noise.RelativeCostOrTime < 0 {
		add("comparison.noise thresholds must be >= 0")
	}

	switch s.SingleAgentSelectionRule {
	case SelectHighestSolo, SelectNamed:
	default:
		add("single_agent_selection_rule must be %s or %s", SelectHighestSolo, SelectNamed)
	}
	if s.SingleAgentSelectionRule == SelectNamed && strings.TrimSpace(s.SingleAgentJustification) == "" {
		add("named single_agent_phenotype_id requires single_agent_justification")
	}

	if s.Tasks != nil {
		errs = append(errs, validateTasks(s)...)
	}
	if s.Pool != nil {
		errs = append(errs, validatePool(s)...)
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("expcontract: %s", strings.Join(errs, "; "))
}

func validateTasks(s *RunSpec) []string {
	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}
	if strings.TrimSpace(s.Tasks.ID) == "" {
		add("task set id is required")
	}
	if s.Tasks.ID != s.Workload.ID {
		add("task set id %q must match workload.id %q", s.Tasks.ID, s.Workload.ID)
	}
	if len(s.Tasks.Tasks) == 0 {
		add("task set is empty")
	}
	ids := map[string]*Task{}
	hasExclusive := false
	for i, t := range s.Tasks.Tasks {
		if strings.TrimSpace(t.ID) == "" {
			add("tasks[%d] missing id", i)
			continue
		}
		if ids[t.ID] != nil {
			add("duplicate task id %q", t.ID)
		}
		ids[t.ID] = &s.Tasks.Tasks[i]
		if !allowedClasses[t.Class] {
			add("task %s: unknown class %q", t.ID, t.Class)
		}
		if t.TimeoutMS < 1 {
			add("task %s: timeout_ms must be >= 1", t.ID)
		}
		if t.MaxAttempts < 1 {
			add("task %s: max_attempts must be >= 1", t.ID)
		}
		if t.Payload == nil && strings.TrimSpace(t.PayloadRef) == "" {
			add("task %s: payload or payload_ref required", t.ID)
		}
		if !allowedVerifiers[t.Verifier.Kind] {
			add("task %s: unknown verifier %q", t.ID, t.Verifier.Kind)
		}
		switch t.Verifier.Kind {
		case VerifierJSONEqual:
			if t.Verifier.Expected == nil && t.Verifier.ExpectedRef == "" {
				add("task %s: deterministic_json_equal needs expected or expected_ref", t.ID)
			}
		case VerifierCommand:
			if len(t.Verifier.Command) == 0 {
				add("task %s: deterministic_command needs command", t.ID)
			}
		case VerifierRubric:
			if t.Verifier.RubricRef == "" {
				add("task %s: rubric_score verifier needs rubric_ref", t.ID)
			}
		case VerifierHuman:
			if t.Verifier.LabelsRef == "" {
				add("task %s: human_label needs labels_ref (labels may not be invented at run time)", t.ID)
			}
		}
		if !allowedQuality[t.QualityOracle.Kind] {
			add("task %s: unknown quality oracle %q", t.ID, t.QualityOracle.Kind)
		}
		if t.QualityOracle.Kind == QualityExplicit && t.QualityOracle.Field == "" {
			add("task %s: explicit_field quality oracle needs field", t.ID)
		}
		if t.ExclusiveSideEffect {
			hasExclusive = true
		}
		switch t.Arrival.Kind {
		case "initial", "on_success_of", "on_failure_of":
		default:
			add("task %s: invalid arrival", t.ID)
		}
	}
	for _, t := range s.Tasks.Tasks {
		if t.Arrival.Kind == "initial" {
			continue
		}
		if _, ok := ids[t.Arrival.TaskID]; !ok {
			add("task %s: arrival references unknown task %q", t.ID, t.Arrival.TaskID)
		}
		if t.Arrival.TaskID == t.ID {
			add("task %s: arrival cannot reference itself", t.ID)
		}
	}
	if hasExclusive && strings.TrimSpace(s.LeaseContractRef) == "" {
		add("exclusive_side_effect tasks require lease_contract_ref (MESH-102)")
	}
	return errs
}

func validatePool(s *RunSpec) []string {
	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}
	if s.Pool.ID == "" {
		add("worker pool id is required")
	}
	if len(s.Pool.Instances) == 0 {
		add("worker pool is empty")
	}
	seenInst := map[string]bool{}
	phenotypes := map[string]bool{}
	foundSingle := false
	for _, inst := range s.Pool.Instances {
		if inst.InstanceID == "" || inst.PhenotypeID == "" {
			add("worker instance needs instance_id and phenotype_id")
			continue
		}
		if seenInst[inst.InstanceID] {
			add("duplicate instance_id %q", inst.InstanceID)
		}
		seenInst[inst.InstanceID] = true
		if len(inst.Capabilities) == 0 {
			add("instance %s has no capabilities", inst.InstanceID)
		}
		phenotypes[inst.PhenotypeID] = true
		if inst.PhenotypeID == s.SingleAgentPhenotypeID {
			foundSingle = true
		}
	}
	if s.SingleAgentPhenotypeID == "" {
		add("single_agent_phenotype_id is required")
	} else if !foundSingle {
		add("single_agent_phenotype_id %q is not in the worker pool", s.SingleAgentPhenotypeID)
	}
	_ = phenotypes
	return errs
}

// HasAllocationSystem reports whether sys is compared with the shared worker pool.
func HasAllocationSystem(sys string) bool {
	return allocationSystems[sys]
}
