package expcontract

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// CanonicalJSON is the locked equality encoding: encoding/json with map keys sorted.
func CanonicalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var norm any
	if err := json.Unmarshal(b, &norm); err != nil {
		return nil, err
	}
	return json.Marshal(norm)
}

// Verify applies the task verifier and quality oracle to an Execute output.
func Verify(t Task, output any) (passed bool, quality float64, err error) {
	switch t.Verifier.Kind {
	case VerifierJSONEqual:
		want := t.Verifier.Expected
		if t.Verifier.ExpectedRef != "" {
			return false, 0, fmt.Errorf("task %s: expected_ref is not resolved in contract v1; inline expected", t.ID)
		}
		passed, err = jsonEqual(output, want)
	default:
		return false, 0, fmt.Errorf("task %s: verifier %q is not executable in contract v1 fixtures", t.ID, t.Verifier.Kind)
	}
	if err != nil {
		return false, 0, err
	}
	switch t.QualityOracle.Kind {
	case QualityBinary:
		if passed {
			quality = 1
		}
	default:
		return passed, 0, fmt.Errorf("task %s: quality oracle %q is not executable in contract v1 fixtures", t.ID, t.QualityOracle.Kind)
	}
	return passed, quality, nil
}

func jsonEqual(a, b any) (bool, error) {
	left, err := CanonicalJSON(a)
	if err != nil {
		return false, err
	}
	right, err := CanonicalJSON(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(left, right), nil
}

// Triggered returns tasks whose arrival condition has occurred.
// passed/attempted are keyed by task id. Initial tasks are always triggered.
func Triggered(tasks []Task, passed, attempted map[string]bool) []Task {
	out := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		switch t.Arrival.Kind {
		case "", "initial":
			out = append(out, t)
		case "on_success_of":
			if passed[t.Arrival.TaskID] {
				out = append(out, t)
			}
		case "on_failure_of":
			if attempted[t.Arrival.TaskID] && !passed[t.Arrival.TaskID] {
				out = append(out, t)
			}
		}
	}
	return out
}
