package expcontract

import (
	"fmt"
	"strings"
)

// MismatchOutput is the locked result when a worker lacks true_capabilities.
var MismatchOutput = map[string]any{"error": "capability_mismatch"}

var allowedTools = map[string]bool{
	ToolJSONAddCount:  true,
	ToolJSONPickKeys:  true,
	ToolJSONMerge:     true,
	ToolJSONSortKeys:  true,
	ToolJSONUnwrapMsg: true,
	ToolJSONWrapText:  true,
	ToolTextClassify:  true,
}

// TrueCapabilities is the execution requirement. Allocators must not read it.
func TrueCapabilities(t Task) []string {
	if len(t.TrueCapabilities) > 0 {
		return t.TrueCapabilities
	}
	return t.RequiredCapabilities
}

// VisibleEligible reports whether a static manager/router may assign this worker
// using only advertised capabilities and the task's visible required_capabilities.
func VisibleEligible(w WorkerInstance, t Task) bool {
	if len(t.RequiredCapabilities) == 0 {
		return true
	}
	return hasAll(w.Capabilities, t.RequiredCapabilities)
}

// WorkerCanExecute reports whether the worker has the true capabilities.
func WorkerCanExecute(w WorkerInstance, t Task) bool {
	return hasAll(w.Capabilities, TrueCapabilities(t))
}

// EffectiveExecMS is task.simulate_exec_ms + worker.simulate_exec_ms.
func EffectiveExecMS(t Task, w WorkerInstance) int {
	return t.SimulateExecMS + w.SimulateExecMS
}

// Execute runs the locked tool. Incapable workers return MismatchOutput.
func Execute(t Task, w WorkerInstance) (any, bool) {
	if !WorkerCanExecute(w, t) {
		return MismatchOutput, false
	}
	out, err := applyTool(t)
	if err != nil {
		return map[string]any{"error": err.Error()}, false
	}
	return out, true
}

func applyTool(t Task) (any, error) {
	payload, _ := t.Payload.(map[string]any)
	if payload == nil && t.Payload != nil {
		return nil, fmt.Errorf("payload must be a JSON object")
	}
	switch t.Tool {
	case ToolJSONAddCount:
		items, _ := payload["items"].([]any)
		out := map[string]any{"items": items, "n": float64(len(items))}
		return out, nil
	case ToolJSONPickKeys:
		keep, _ := payload["keep"].([]any)
		out := map[string]any{}
		for _, k := range keep {
			ks, _ := k.(string)
			if v, ok := payload[ks]; ok {
				out[ks] = v
			}
		}
		return out, nil
	case ToolJSONMerge:
		left, _ := payload["left"].(map[string]any)
		right, _ := payload["right"].(map[string]any)
		out := map[string]any{}
		for k, v := range left {
			out[k] = v
		}
		for k, v := range right {
			out[k] = v
		}
		return out, nil
	case ToolJSONSortKeys:
		keys, _ := payload["keys"].([]any)
		strs := make([]string, 0, len(keys))
		for _, k := range keys {
			s, _ := k.(string)
			strs = append(strs, s)
		}
		for i := 0; i < len(strs); i++ {
			for j := i + 1; j < len(strs); j++ {
				if strs[j] < strs[i] {
					strs[i], strs[j] = strs[j], strs[i]
				}
			}
		}
		outKeys := make([]any, len(strs))
		for i, s := range strs {
			outKeys[i] = s
		}
		return map[string]any{"keys": outKeys}, nil
	case ToolJSONUnwrapMsg:
		raw, _ := payload["raw"].(map[string]any)
		return map[string]any{"text": raw["msg"]}, nil
	case ToolJSONWrapText:
		return map[string]any{"wrapped": true, "n": float64(1)}, nil
	case ToolTextClassify:
		text, _ := payload["text"].(string)
		return map[string]any{"label": classify(text)}, nil
	default:
		return nil, fmt.Errorf("unknown tool %q", t.Tool)
	}
}

func classify(text string) string {
	s := strings.ToLower(text)
	switch {
	case strings.Contains(s, "disk") || strings.Contains(s, "/var"):
		return "infra"
	case strings.Contains(s, "token") || strings.Contains(s, "user"):
		return "auth"
	case strings.Contains(s, "invoice") || strings.Contains(s, "unpaid"):
		return "billing"
	default:
		return "other"
	}
}

func hasAll(have, need []string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[h] = true
	}
	for _, n := range need {
		if !set[n] {
			return false
		}
	}
	return true
}

func sameSet(a, b []string) bool {
	return hasAll(a, b) && hasAll(b, a)
}

// UncertainFraction is the share of tasks whose visible labels are empty or
// disagree with true_capabilities.
func UncertainFraction(tasks []Task) float64 {
	if len(tasks) == 0 {
		return 0
	}
	u := 0
	for _, t := range tasks {
		if len(t.RequiredCapabilities) == 0 || !sameSet(t.RequiredCapabilities, TrueCapabilities(t)) {
			u++
		}
	}
	return float64(u) / float64(len(tasks))
}
