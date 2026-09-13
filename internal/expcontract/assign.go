package expcontract

import "sort"

// AssignStatic is the locked assignment policy for manager_worker and
// central_router. It uses only visible required_capabilities.
//
// Among idle workers that are VisibleEligible: fewest capabilities
// (most specific), then lowest simulate_exec_ms, then instance_id.
// If none are eligible, the task stays queued (ok=false). The baseline
// does not reassign on failure.
func AssignStatic(task Task, idle []WorkerInstance) (WorkerInstance, bool) {
	cand := make([]WorkerInstance, 0, len(idle))
	for _, w := range idle {
		if VisibleEligible(w, task) {
			cand = append(cand, w)
		}
	}
	if len(cand) == 0 {
		return WorkerInstance{}, false
	}
	sort.Slice(cand, func(i, j int) bool {
		if len(cand[i].Capabilities) != len(cand[j].Capabilities) {
			return len(cand[i].Capabilities) < len(cand[j].Capabilities)
		}
		if cand[i].SimulateExecMS != cand[j].SimulateExecMS {
			return cand[i].SimulateExecMS < cand[j].SimulateExecMS
		}
		return cand[i].InstanceID < cand[j].InstanceID
	})
	return cand[0], true
}
