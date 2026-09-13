// Package expcontract validates the v2 experiment contract (MESH-101).
// It does not run benchmarks or implement allocation.
package expcontract

const Version = 1

const (
	SystemSingleAgent    = "single_agent"
	SystemManagerWorker  = "manager_worker"
	SystemCentralRouter  = "central_router"
	SystemMeshyAnts      = "meshyants"
	SystemFlatBlackboard = "flat_blackboard"
)

const (
	ClassHomogeneousDAG         = "homogeneous_dag"
	ClassHeterogeneousUncertain = "heterogeneous_uncertain"
	ClassPerturbation           = "perturbation"
	ClassDurableDAG             = "durable_dag"
)

const (
	VerifierJSONEqual = "deterministic_json_equal"
	VerifierCommand   = "deterministic_command"
	VerifierRubric    = "rubric_score"
	VerifierHuman     = "human_label"
)

const (
	QualityBinary   = "binary_from_verifier"
	QualityRubric   = "rubric_score"
	QualityExplicit = "explicit_field"
)

const (
	MetricSuccessRate     = "task_success_rate"
	MetricQuality         = "quality"
	MetricWallTime        = "wall_time_ms"
	MetricCost            = "cost_micros"
	MetricModelCalls      = "model_calls"
	MetricInputTokens     = "input_tokens"
	MetricOutputTokens    = "output_tokens"
	MetricDuplicateWork   = "duplicate_work_rate"
	MetricWakeups         = "wakeup_count"
	MetricFailedClaims    = "failed_claim_count"
	MetricAllocMessages   = "allocation_message_count"
	MetricCoordModelCalls = "coordination_model_calls"
	MetricRecoveryTime    = "recovery_time_ms"
	MetricEmergenceDelta  = "emergence_delta"
	WinRulePairedSign80   = "paired_sign_80"
	WinRuleBootstrapCI95  = "bootstrap_ci_95"
	SelectHighestSolo     = "highest_solo_success"
	SelectNamed           = "named"
)

var requiredSystems = []string{
	SystemSingleAgent,
	SystemManagerWorker,
	SystemCentralRouter,
	SystemMeshyAnts,
}

var requiredPrimaryMetrics = []string{
	MetricSuccessRate,
	MetricQuality,
	MetricWallTime,
	MetricCost,
	MetricModelCalls,
}

var requiredSecondaryMetrics = []string{
	MetricDuplicateWork,
	MetricWakeups,
	MetricFailedClaims,
	MetricCoordModelCalls,
}

var allowedSystems = map[string]bool{
	SystemSingleAgent:    true,
	SystemManagerWorker:  true,
	SystemCentralRouter:  true,
	SystemMeshyAnts:      true,
	SystemFlatBlackboard: true,
}

var allowedClasses = map[string]bool{
	ClassHomogeneousDAG:         true,
	ClassHeterogeneousUncertain: true,
	ClassPerturbation:           true,
	ClassDurableDAG:             true,
}

var allowedVerifiers = map[string]bool{
	VerifierJSONEqual: true,
	VerifierCommand:   true,
	VerifierRubric:    true,
	VerifierHuman:     true,
}

var allowedQuality = map[string]bool{
	QualityBinary:   true,
	QualityRubric:   true,
	QualityExplicit: true,
}

var allowedMetrics = map[string]bool{
	MetricSuccessRate:     true,
	MetricQuality:         true,
	MetricWallTime:        true,
	MetricCost:            true,
	MetricModelCalls:      true,
	MetricInputTokens:     true,
	MetricOutputTokens:    true,
	MetricDuplicateWork:   true,
	MetricWakeups:         true,
	MetricFailedClaims:    true,
	MetricAllocMessages:   true,
	MetricCoordModelCalls: true,
	MetricRecoveryTime:    true,
	MetricEmergenceDelta:  true,
}

var allocationSystems = map[string]bool{
	SystemManagerWorker:  true,
	SystemCentralRouter:  true,
	SystemMeshyAnts:      true,
	SystemFlatBlackboard: true,
}

// RunSpec is a locked four-arm (or five-arm) experiment description.
type RunSpec struct {
	ContractVersion               int              `json:"contract_version"`
	RunID                         string           `json:"run_id"`
	Phase                         string           `json:"phase"`
	Hypothesis                    string           `json:"hypothesis"`
	Falsification                 string           `json:"falsification"`
	Workload                      WorkloadRef      `json:"workload"`
	WorkerPoolRef                 string           `json:"worker_pool_ref"`
	SingleAgentPhenotypeID        string           `json:"single_agent_phenotype_id"`
	SingleAgentSelectionRule      string           `json:"single_agent_selection_rule"`
	SingleAgentJustification      string           `json:"single_agent_justification,omitempty"`
	LeaseContractRef              string           `json:"lease_contract_ref,omitempty"`
	Tools                         []string         `json:"tools"`
	Models                        []Model          `json:"models"`
	Tokenizer                     string           `json:"tokenizer,omitempty"`
	PriceBook                     map[string]Price `json:"price_book,omitempty"`
	Budget                        Budget           `json:"budget"`
	Systems                       []string         `json:"systems"`
	Repeats                       int              `json:"repeats"`
	Seeds                         []int            `json:"seeds"`
	RNGAlgorithm                  string           `json:"rng_algorithm"`
	FailureInjection              any              `json:"failure_injection"`
	Metrics                       []string         `json:"metrics"`
	Comparison                    Comparison       `json:"comparison"`
	AllowMeshyantsCoordinationLLM bool             `json:"allow_meshyants_coordination_llm,omitempty"`

	Tasks *TaskSet    `json:"-"`
	Pool  *WorkerPool `json:"-"`
}

type WorkloadRef struct {
	ID                     string `json:"id"`
	Class                  string `json:"class"`
	TasksRef               string `json:"tasks_ref"`
	DecisiveSwarmBenchmark bool   `json:"decisive_swarm_benchmark"`
}

type Model struct {
	ID               string `json:"id"`
	Provider         string `json:"provider"`
	MaxContextTokens int    `json:"max_context_tokens,omitempty"`
}

type Price struct {
	InputMicrosPer1k  int `json:"input_micros_per_1k"`
	OutputMicrosPer1k int `json:"output_micros_per_1k"`
}

type Budget struct {
	MaxWallMS                 int `json:"max_wall_ms"`
	MaxModelCalls             int `json:"max_model_calls"`
	MaxInputTokens            int `json:"max_input_tokens"`
	MaxOutputTokens           int `json:"max_output_tokens"`
	MaxCostMicros             int `json:"max_cost_micros"`
	MaxWorkerWakeups          int `json:"max_worker_wakeups"`
	MaxExclusiveLeaseAttempts int `json:"max_exclusive_lease_attempts"`
}

type Comparison struct {
	WinRule string `json:"win_rule"`
	Noise   Noise  `json:"noise"`
}

type Noise struct {
	TaskSuccessRate    float64 `json:"task_success_rate"`
	Quality            float64 `json:"quality"`
	RelativeCostOrTime float64 `json:"relative_cost_or_time"`
	DuplicateWorkRate  float64 `json:"duplicate_work_rate"`
}

type TaskSet struct {
	ID    string `json:"id"`
	Tasks []Task `json:"tasks"`
}

type Task struct {
	ID                   string   `json:"id"`
	Class                string   `json:"class"`
	RequiredCapabilities []string `json:"required_capabilities"`
	Arrival              Arrival  `json:"arrival"`
	ExclusiveSideEffect  bool     `json:"exclusive_side_effect"`
	TimeoutMS            int      `json:"timeout_ms"`
	MaxAttempts          int      `json:"max_attempts"`
	Payload              any      `json:"payload,omitempty"`
	PayloadRef           string   `json:"payload_ref,omitempty"`
	SimulateExecMS       int      `json:"simulate_exec_ms,omitempty"`
	Verifier             Verifier `json:"verifier"`
	QualityOracle        Quality  `json:"quality_oracle"`
}

type Arrival struct {
	Kind   string // "initial", "on_success_of", "on_failure_of"
	TaskID string
}

type Verifier struct {
	Kind        string   `json:"kind"`
	Expected    any      `json:"expected,omitempty"`
	ExpectedRef string   `json:"expected_ref,omitempty"`
	Command     []string `json:"command,omitempty"`
	StdoutMatch string   `json:"stdout_match,omitempty"`
	RubricRef   string   `json:"rubric_ref,omitempty"`
	LabelsRef   string   `json:"labels_ref,omitempty"`
}

type Quality struct {
	Kind      string `json:"kind"`
	Field     string `json:"field,omitempty"`
	RubricRef string `json:"rubric_ref,omitempty"`
}

type WorkerPool struct {
	ID        string           `json:"id"`
	Instances []WorkerInstance `json:"instances"`
}

type WorkerInstance struct {
	InstanceID     string   `json:"instance_id"`
	PhenotypeID    string   `json:"phenotype_id"`
	Capabilities   []string `json:"capabilities"`
	SimulateExecMS int      `json:"simulate_exec_ms,omitempty"`
}
