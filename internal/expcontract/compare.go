package expcontract

import (
	"fmt"
	"math"
)

const (
	VerdictAdvantage    = "meshyants_advantage"
	VerdictFalsified    = "falsified"
	VerdictMixed        = "mixed"
	VerdictInconclusive = "inconclusive"
	DecisionSignificant = "significant"
	DecisionTie         = "tie"
	GroupAllocation     = "allocation"
	GroupEmergence      = "emergence"
)

var higherBetter = map[string]bool{
	MetricSuccessRate: true,
	MetricQuality:     true,
}

var allocationPrimary = []string{
	MetricSuccessRate,
	MetricQuality,
	MetricCost,
	MetricWallTime,
	MetricModelCalls,
}

// ArmRepeat is one system's result on one seed. MESH-103 fills this from a run-result file.
type ArmRepeat struct {
	System            string
	Seed              int
	BudgetExceeded    bool
	ProtocolViolation string
	Metrics           map[string]float64
}

// MetricDecision is the pairwise-or-group outcome for one metric.
type MetricDecision struct {
	Group    string             `json:"group"`
	Winner   string             `json:"winner"`
	Decision string             `json:"decision"`
	Means    map[string]float64 `json:"means"`
}

// ComparisonReport is the machine-readable Phase A verdict.
type ComparisonReport struct {
	ContractVersion    int                       `json:"contract_version"`
	RunID              string                    `json:"run_id"`
	Repeats            int                       `json:"repeats"`
	Seeds              []int                     `json:"seeds"`
	AllocationVerdict  string                    `json:"allocation_verdict"`
	EmergenceDeltaMean *float64                  `json:"emergence_delta_mean"`
	PerMetric          map[string]MetricDecision `json:"per_metric"`
	Notes              string                    `json:"notes,omitempty"`
}

// Compare implements the locked win rule. It does not accept per-arm budget overrides;
// callers must already have validated the spec.
func Compare(spec *RunSpec, rows []ArmRepeat) (*ComparisonReport, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil spec")
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	bySysSeed := map[string]map[int]ArmRepeat{}
	for _, row := range rows {
		if bySysSeed[row.System] == nil {
			bySysSeed[row.System] = map[int]ArmRepeat{}
		}
		bySysSeed[row.System][row.Seed] = row
	}
	for _, sys := range spec.Systems {
		for _, seed := range spec.Seeds {
			row, ok := bySysSeed[sys][seed]
			if !ok {
				return nil, fmt.Errorf("missing result for system %s seed %d", sys, seed)
			}
			if sys == SystemMeshyAnts && row.ProtocolViolation != "" {
				return &ComparisonReport{
					ContractVersion:   Version,
					RunID:             spec.RunID,
					Repeats:           spec.Repeats,
					Seeds:             append([]int(nil), spec.Seeds...),
					AllocationVerdict: VerdictInconclusive,
					PerMetric:         map[string]MetricDecision{},
					Notes:             "meshyants protocol_violation; verdict is not usable",
				}, nil
			}
		}
	}

	report := &ComparisonReport{
		ContractVersion: Version,
		RunID:           spec.RunID,
		Repeats:         spec.Repeats,
		Seeds:           append([]int(nil), spec.Seeds...),
		PerMetric:       map[string]MetricDecision{},
	}

	// Emergence delta: quality(meshyants) - quality(single_agent), mean over seeds.
	if q := pairedValues(spec, bySysSeed, SystemMeshyAnts, MetricQuality); len(q) == spec.Repeats {
		if s := pairedValues(spec, bySysSeed, SystemSingleAgent, MetricQuality); len(s) == spec.Repeats {
			sum := 0.0
			for i := range q {
				sum += q[i] - s[i]
			}
			mean := sum / float64(len(q))
			report.EmergenceDeltaMean = &mean
			winner := DecisionTie
			side := "tie"
			if mean > spec.Comparison.Noise.Quality {
				winner = DecisionSignificant
				side = SystemMeshyAnts
			} else if mean < -spec.Comparison.Noise.Quality {
				winner = DecisionSignificant
				side = SystemSingleAgent
			}
			report.PerMetric[MetricEmergenceDelta] = MetricDecision{
				Group:    GroupEmergence,
				Winner:   side,
				Decision: winner,
				Means: map[string]float64{
					SystemMeshyAnts:   meanOf(q),
					SystemSingleAgent: meanOf(s),
				},
			}
		}
	}

	betterCount := 0
	worseCount := 0
	// Falsified if either simpler required system matches/beats meshyants on every primary.
	managerDominates := true
	routerDominates := true

	for _, metric := range allocationPrimary {
		mesh := pairedValues(spec, bySysSeed, SystemMeshyAnts, metric)
		mgr := pairedValues(spec, bySysSeed, SystemManagerWorker, metric)
		rtr := pairedValues(spec, bySysSeed, SystemCentralRouter, metric)
		if len(mesh) != spec.Repeats || len(mgr) != spec.Repeats || len(rtr) != spec.Repeats {
			return nil, fmt.Errorf("incomplete primary metric %s", metric)
		}
		absN, relN := noiseFor(spec, metric)
		vsMgr := decidePair(mesh, mgr, higherBetter[metric], absN, relN, spec)
		vsRtr := decidePair(mesh, rtr, higherBetter[metric], absN, relN, spec)

		winner := "tie"
		decision := DecisionTie
		if vsMgr.meshBetter && vsRtr.meshBetter {
			winner = SystemMeshyAnts
			decision = DecisionSignificant
			betterCount++
		} else if vsMgr.otherBetter || vsRtr.otherBetter {
			if vsMgr.otherBetter && vsRtr.otherBetter {
				winner = "simpler_systems"
			} else if vsMgr.otherBetter {
				winner = SystemManagerWorker
			} else {
				winner = SystemCentralRouter
			}
			decision = DecisionSignificant
			worseCount++
		}
		if vsMgr.meshBetter || vsRtr.otherBetter {
			// mixed pairwise on this metric; keep the aggregate winner above
		}

		report.PerMetric[metric] = MetricDecision{
			Group:    GroupAllocation,
			Winner:   winner,
			Decision: decision,
			Means: map[string]float64{
				SystemMeshyAnts:     meanOf(mesh),
				SystemManagerWorker: meanOf(mgr),
				SystemCentralRouter: meanOf(rtr),
			},
		}

		// Dominates = other is tie or better on this metric (not meshBetter).
		if vsMgr.meshBetter {
			managerDominates = false
		}
		if vsRtr.meshBetter {
			routerDominates = false
		}
	}

	switch {
	case (managerDominates || routerDominates) && worseCount >= 1:
		// A simpler arm is tied-or-better on every primary and significantly
		// better on at least one. All-tie is inconclusive, not falsified.
		report.AllocationVerdict = VerdictFalsified
	case betterCount >= 1 && worseCount == 0:
		report.AllocationVerdict = VerdictAdvantage
	case betterCount >= 1 && worseCount >= 1:
		report.AllocationVerdict = VerdictMixed
	default:
		report.AllocationVerdict = VerdictInconclusive
	}
	return report, nil
}

type pairResult struct {
	meshBetter  bool
	otherBetter bool
}

func decidePair(mesh, other []float64, higher bool, absNoise, relNoise float64, spec *RunSpec) pairResult {
	n := len(mesh)
	diff := make([]float64, n)
	for i := 0; i < n; i++ {
		if higher {
			diff[i] = mesh[i] - other[i]
		} else {
			diff[i] = other[i] - mesh[i]
		}
	}
	meanDiff := meanOf(diff)
	meshMean := meanOf(mesh)
	otherMean := meanOf(other)
	betterMean := meshMean
	if higher {
		if otherMean > meshMean {
			betterMean = otherMean
		}
	} else {
		betterMean = math.Min(meshMean, otherMean)
	}
	noise := absNoise
	if absNoise == 0 {
		noise = relNoise * math.Abs(betterMean)
	}
	pos, neg := 0, 0
	for _, d := range diff {
		if d > 0 {
			pos++
		} else if d < 0 {
			neg++
		}
	}
	need := int(math.Ceil(0.8 * float64(spec.Repeats)))
	if spec.Comparison.WinRule == WinRuleBootstrapCI95 {
		// Same information with n=3; treat as paired-sign until MESH-103 records enough repeats.
		need = int(math.Ceil(0.8 * float64(spec.Repeats)))
	}
	// |diff| <= noise is a tie, including float error on the threshold.
	const eps = 1e-9
	switch {
	case meanDiff > noise+eps && pos >= need:
		return pairResult{meshBetter: true}
	case meanDiff < -(noise+eps) && neg >= need:
		return pairResult{otherBetter: true}
	default:
		return pairResult{}
	}
}

func noiseFor(spec *RunSpec, metric string) (abs, rel float64) {
	switch metric {
	case MetricSuccessRate:
		return spec.Comparison.Noise.TaskSuccessRate, 0
	case MetricQuality:
		return spec.Comparison.Noise.Quality, 0
	default:
		return 0, spec.Comparison.Noise.RelativeCostOrTime
	}
}

func pairedValues(spec *RunSpec, bySysSeed map[string]map[int]ArmRepeat, sys, metric string) []float64 {
	out := make([]float64, 0, spec.Repeats)
	for _, seed := range spec.Seeds {
		row, ok := bySysSeed[sys][seed]
		if !ok {
			return nil
		}
		if row.BudgetExceeded && (metric == MetricSuccessRate || metric == MetricQuality) {
			out = append(out, 0)
			continue
		}
		v, ok := row.Metrics[metric]
		if !ok {
			return nil
		}
		out = append(out, v)
	}
	return out
}

func meanOf(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}
