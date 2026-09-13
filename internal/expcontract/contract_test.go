package expcontract_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/meshyants/meshyants/v1/internal/expcontract"
	"github.com/stretchr/testify/require"
)

func exampleRunPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "docs", "experiments", "examples", "phase-a-synthetic", "run.json")
}

func TestExampleRunSpecValidates(t *testing.T) {
	t.Parallel()
	spec, err := expcontract.Load(exampleRunPath(t))
	require.NoError(t, err)
	require.Equal(t, "phase-a-synthetic-v1", spec.RunID)
	require.Equal(t, 4, len(spec.Systems))
	require.Equal(t, 12, len(spec.Tasks.Tasks))
	require.Equal(t, 5, len(spec.Pool.Instances))
	require.Equal(t, "generalist", spec.SingleAgentPhenotypeID)
	require.False(t, spec.Workload.DecisiveSwarmBenchmark)
	require.Equal(t, expcontract.ClassHomogeneousDAG, spec.Workload.Class)
}

func TestUncertainFixtureValidates(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "..", "..", "docs", "experiments", "examples", "phase-a-uncertain", "run.json")
	spec, err := expcontract.Load(path)
	require.NoError(t, err)
	require.True(t, spec.Workload.DecisiveSwarmBenchmark)
	require.GreaterOrEqual(t, expcontract.UncertainFraction(spec.Tasks.Tasks), 0.25)
}

func TestRejectsDecisiveLabeledDAG(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.Workload.DecisiveSwarmBenchmark = true
	require.Error(t, spec.Validate())
}

func TestRejectsDurableDAGAsDecisive(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.Workload.Class = expcontract.ClassDurableDAG
	spec.Workload.DecisiveSwarmBenchmark = true
	require.Error(t, spec.Validate())
}

func TestRejectsMissingSystem(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.Systems = []string{
		expcontract.SystemSingleAgent,
		expcontract.SystemManagerWorker,
		expcontract.SystemMeshyAnts,
	}
	require.Error(t, spec.Validate())
}

func TestRejectsCoordinationLLM(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.AllowMeshyantsCoordinationLLM = true
	require.Error(t, spec.Validate())
}

func TestRejectsNamedSingleAgentWithoutJustification(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.SingleAgentJustification = ""
	require.Error(t, spec.Validate())
}

func TestRejectsSingleAgentNotInPool(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.SingleAgentPhenotypeID = "secret-super-agent"
	require.Error(t, spec.Validate())
}

func TestRejectsExclusiveWithoutLease(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.Tasks.Tasks[0].ExclusiveSideEffect = true
	spec.LeaseContractRef = ""
	require.Error(t, spec.Validate())
}

func TestRejectsUnknownMetric(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.Metrics = append(spec.Metrics, "tasks_per_hour")
	require.Error(t, spec.Validate())
}

func TestRejectsSeedRepeatMismatch(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.Repeats = 4
	require.Error(t, spec.Validate())
}

func TestRejectsLLMRunWithoutTokenizer(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.Models = []expcontract.Model{{ID: "m", Provider: "x"}}
	spec.Budget.MaxModelCalls = 10
	spec.Budget.MaxInputTokens = 100
	spec.Budget.MaxOutputTokens = 100
	spec.Budget.MaxCostMicros = 1
	require.Error(t, spec.Validate())
}

func TestRejectsUnknownArrivalParent(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.Tasks.Tasks[9].Arrival = expcontract.Arrival{Kind: "on_success_of", TaskID: "no-such-task"}
	require.Error(t, spec.Validate())
}

func TestCompareAllTieIsInconclusive(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	rows := evenRows(spec, 1.0, 1.0, 0, 100, 0)
	report, err := expcontract.Compare(spec, rows)
	require.NoError(t, err)
	require.Equal(t, expcontract.VerdictInconclusive, report.AllocationVerdict)
	require.NotNil(t, report.EmergenceDeltaMean)
	require.Equal(t, 0.0, *report.EmergenceDeltaMean)
}

func TestCompareFalsifiedWhenRouterBeatsQuality(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	rows := evenRows(spec, 1.0, 0.7, 0, 100, 0)
	for i := range rows {
		if rows[i].System == expcontract.SystemCentralRouter || rows[i].System == expcontract.SystemManagerWorker {
			rows[i].Metrics[expcontract.MetricQuality] = 0.95
		}
	}
	report, err := expcontract.Compare(spec, rows)
	require.NoError(t, err)
	require.Equal(t, expcontract.VerdictFalsified, report.AllocationVerdict)
}

func TestCompareMeshyAntsAdvantageOnQuality(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	rows := evenRows(spec, 1.0, 0.7, 0, 100, 0)
	for _, seed := range spec.Seeds {
		for j := range rows {
			if rows[j].System == expcontract.SystemMeshyAnts && rows[j].Seed == seed {
				rows[j].Metrics[expcontract.MetricQuality] = 0.9
			}
		}
	}
	report, err := expcontract.Compare(spec, rows)
	require.NoError(t, err)
	require.Equal(t, expcontract.VerdictAdvantage, report.AllocationVerdict)
	require.Greater(t, *report.EmergenceDeltaMean, 0.0)
}

func TestCompareProtocolViolationIsInconclusive(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	rows := evenRows(spec, 1.0, 1.0, 0, 100, 0)
	for i := range rows {
		if rows[i].System == expcontract.SystemMeshyAnts && rows[i].Seed == spec.Seeds[0] {
			rows[i].ProtocolViolation = "coordination_model_calls>0"
		}
	}
	report, err := expcontract.Compare(spec, rows)
	require.NoError(t, err)
	require.Equal(t, expcontract.VerdictInconclusive, report.AllocationVerdict)
}

func TestCompareNoiseThresholdIsTie(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	rows := evenRows(spec, 1.0, 0.70, 0, 100, 0)
	for i := range rows {
		if rows[i].System == expcontract.SystemMeshyAnts {
			rows[i].Metrics[expcontract.MetricQuality] = 0.75 // exactly +0.05, the noise threshold
		}
	}
	report, err := expcontract.Compare(spec, rows)
	require.NoError(t, err)
	require.Equal(t, expcontract.VerdictInconclusive, report.AllocationVerdict)
	require.Equal(t, "tie", report.PerMetric[expcontract.MetricQuality].Decision)
}

func TestCompareBudgetExceededZerosQuality(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	rows := evenRows(spec, 1.0, 1.0, 0, 100, 0)
	for i := range rows {
		if rows[i].System == expcontract.SystemMeshyAnts {
			rows[i].BudgetExceeded = true
		}
	}
	report, err := expcontract.Compare(spec, rows)
	require.NoError(t, err)
	require.Equal(t, expcontract.VerdictFalsified, report.AllocationVerdict)
	require.NotNil(t, report.EmergenceDeltaMean)
	require.Equal(t, -1.0, *report.EmergenceDeltaMean)
}

func loadClone(t *testing.T) *expcontract.RunSpec {
	t.Helper()
	spec, err := expcontract.Load(exampleRunPath(t))
	require.NoError(t, err)
	return spec
}

func TestCompareDecisiveAllTieIsFalsified(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "..", "..", "docs", "experiments", "examples", "phase-a-uncertain", "run.json")
	spec, err := expcontract.Load(path)
	require.NoError(t, err)
	rows := evenRows(spec, 0.7, 0.7, 0, 100, 0)
	report, err := expcontract.Compare(spec, rows)
	require.NoError(t, err)
	require.Equal(t, expcontract.VerdictFalsified, report.AllocationVerdict)
}

func TestCompareMissingSecondaryErrors(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	rows := evenRows(spec, 1.0, 1.0, 0, 100, 0)
	for i := range rows {
		if rows[i].System == expcontract.SystemMeshyAnts {
			delete(rows[i].Metrics, expcontract.MetricAllocMessages)
		}
	}
	_, err := expcontract.Compare(spec, rows)
	require.Error(t, err)
}

func TestRejectsUnknownFailureInjection(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	spec.FailureInjection = &expcontract.FailureInjection{Kind: "kill_worker"}
	require.Error(t, spec.Validate())
}

func TestLockedToolsMatchExpected(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	generalist := spec.Pool.Instances[4]
	require.Equal(t, "generalist", generalist.PhenotypeID)
	for _, task := range spec.Tasks.Tasks {
		out, ok := expcontract.Execute(task, generalist)
		require.True(t, ok, task.ID)
		require.Equal(t, task.Verifier.Expected, out, task.ID)
	}
}

func TestMismatchWhenWorkerLacksTrueCapability(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	jsonWorker := spec.Pool.Instances[0]
	var textTask expcontract.Task
	for _, task := range spec.Tasks.Tasks {
		if task.Tool == expcontract.ToolTextClassify {
			textTask = task
			break
		}
	}
	out, ok := expcontract.Execute(textTask, jsonWorker)
	require.False(t, ok)
	require.Equal(t, expcontract.MismatchOutput, out)
}

func TestAssignStaticPrefersSpecialist(t *testing.T) {
	t.Parallel()
	spec := loadClone(t)
	var jsonTask expcontract.Task
	for _, task := range spec.Tasks.Tasks {
		if task.Tool == expcontract.ToolJSONAddCount {
			jsonTask = task
			break
		}
	}
	got, ok := expcontract.AssignStatic(jsonTask, spec.Pool.Instances)
	require.True(t, ok)
	require.Equal(t, "json-a", got.InstanceID)
}

func TestAssignStaticFollowsWrongVisibleLabel(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	path := filepath.Join(filepath.Dir(file), "..", "..", "docs", "experiments", "examples", "phase-a-uncertain", "run.json")
	spec, err := expcontract.Load(path)
	require.NoError(t, err)
	var merge expcontract.Task
	for _, task := range spec.Tasks.Tasks {
		if task.ID == "json-merge" {
			merge = task
			break
		}
	}
	got, ok := expcontract.AssignStatic(merge, spec.Pool.Instances)
	require.True(t, ok)
	require.Equal(t, "text-a", got.InstanceID)
	_, execOK := expcontract.Execute(merge, got)
	require.False(t, execOK)
}

func evenRows(spec *expcontract.RunSpec, success, quality, cost, wall, calls float64) []expcontract.ArmRepeat {
	var rows []expcontract.ArmRepeat
	for _, sys := range spec.Systems {
		for _, seed := range spec.Seeds {
			rows = append(rows, expcontract.ArmRepeat{
				System: sys,
				Seed:   seed,
				Metrics: map[string]float64{
					expcontract.MetricSuccessRate:     success,
					expcontract.MetricQuality:         quality,
					expcontract.MetricCost:            cost,
					expcontract.MetricWallTime:        wall,
					expcontract.MetricModelCalls:      calls,
					expcontract.MetricDuplicateWork:   0,
					expcontract.MetricWakeups:         12,
					expcontract.MetricFailedClaims:    0,
					expcontract.MetricAllocMessages:   12,
					expcontract.MetricCoordModelCalls: 0,
				},
			})
		}
	}
	return rows
}
