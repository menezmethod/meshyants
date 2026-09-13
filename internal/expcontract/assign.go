package expcontract

import "sort"

// Assignment is one locked schedule row used to compute simulated makespan.
type Assignment struct {
	Task    Task
	Worker  WorkerInstance
	Attempt int
	Passed  bool
	StartMS int
	ExecMS  int
}

// AssignStatic picks among idle VisibleEligible workers: fewest capabilities,
// then lowest simulate_exec_ms, then instance_id. If none, the task stays queued.
func AssignStatic(task Task, idle []WorkerInstance) (WorkerInstance, bool) {
	return AssignStaticRetry(task, idle, nil)
}

// AssignStaticRetry is the required manager_worker and central_router policy.
// It is AssignStatic with failed instance IDs removed. After capability_mismatch,
// those arms MUST exclude the failed worker and pick again (up to max_attempts).
func AssignStaticRetry(task Task, idle []WorkerInstance, exclude map[string]bool) (WorkerInstance, bool) {
	cand := make([]WorkerInstance, 0, len(idle))
	for _, w := range idle {
		if exclude[w.InstanceID] {
			continue
		}
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

// RunStaticRetry executes one task with the required baseline policy.
// The full remaining pool is treated as idle; parallelism is a harness
// concern for wall time only and must not change who is eligible.
func RunStaticRetry(task Task, pool []WorkerInstance) (output any, passed bool, attempts []Assignment) {
	exclude := map[string]bool{}
	for n := 1; n <= task.MaxAttempts; n++ {
		w, ok := AssignStaticRetry(task, pool, exclude)
		if !ok {
			return nil, false, attempts
		}
		out, execOK := Execute(task, w)
		okPass := false
		if execOK {
			var err error
			okPass, _, err = Verify(task, out)
			if err != nil {
				okPass = false
			}
		}
		attempts = append(attempts, Assignment{Task: task, Worker: w, Attempt: n, Passed: okPass, ExecMS: EffectiveExecMS(task, w)})
		if okPass {
			return out, true, attempts
		}
		exclude[w.InstanceID] = true
		output = out
	}
	return output, false, attempts
}

// SimulateStaticRetry runs the locked baseline over a spec: triggered tasks
// in id order, RunStaticRetry, then recompute triggers. This is the success
// and quality the required simpler arms must be able to reproduce.
func SimulateStaticRetry(spec *RunSpec) (successRate, quality float64, assignments []Assignment) {
	if spec == nil || spec.Tasks == nil || spec.Pool == nil {
		return 0, 0, nil
	}
	passed := map[string]bool{}
	attempted := map[string]bool{}
	done := map[string]bool{}
	var qSum float64
	triggeredN := 0
	for {
		ready := Triggered(spec.Tasks.Tasks, passed, attempted)
		sort.Slice(ready, func(i, j int) bool { return ready[i].ID < ready[j].ID })
		progress := false
		for _, t := range ready {
			if done[t.ID] {
				continue
			}
			_, ok, atts := RunStaticRetry(t, spec.Pool.Instances)
			assignments = append(assignments, atts...)
			attempted[t.ID] = true
			done[t.ID] = true
			passed[t.ID] = ok
			triggeredN++
			if ok {
				qSum++
			}
			progress = true
			break
		}
		if !progress {
			break
		}
	}
	if triggeredN == 0 {
		return 0, 0, assignments
	}
	n := float64(triggeredN)
	return float64(len(passedTrue(passed))) / n, qSum / n, assignments
}

func passedTrue(passed map[string]bool) []string {
	var ids []string
	for id, ok := range passed {
		if ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// MakespanMS is the locked discrete clock: each worker is busy for ExecMS
// after the previous assignment on that worker. Process wall clock must not
// be used as wall_time_ms for these fixtures.
func MakespanMS(assignments []Assignment) int {
	busy := map[string]int{}
	maxT := 0
	for i := range assignments {
		w := assignments[i].Worker.InstanceID
		start := busy[w]
		end := start + assignments[i].ExecMS
		busy[w] = end
		assignments[i].StartMS = start
		if end > maxT {
			maxT = end
		}
	}
	return maxT
}
