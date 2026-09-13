package expcontract

import (
	"encoding/json"
	"fmt"
)

func (a *Arrival) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		if s != "initial" {
			return fmt.Errorf("arrival string must be %q", "initial")
		}
		*a = Arrival{Kind: "initial"}
		return nil
	}
	var obj map[string]string
	if err := json.Unmarshal(b, &obj); err != nil {
		return fmt.Errorf("arrival: %w", err)
	}
	if id, ok := obj["on_success_of"]; ok && id != "" && len(obj) == 1 {
		*a = Arrival{Kind: "on_success_of", TaskID: id}
		return nil
	}
	if id, ok := obj["on_failure_of"]; ok && id != "" && len(obj) == 1 {
		*a = Arrival{Kind: "on_failure_of", TaskID: id}
		return nil
	}
	return fmt.Errorf("arrival must be \"initial\" or {on_success_of|on_failure_of}")
}

func (a Arrival) MarshalJSON() ([]byte, error) {
	switch a.Kind {
	case "", "initial":
		return json.Marshal("initial")
	case "on_success_of":
		return json.Marshal(map[string]string{"on_success_of": a.TaskID})
	case "on_failure_of":
		return json.Marshal(map[string]string{"on_failure_of": a.TaskID})
	default:
		return nil, fmt.Errorf("unknown arrival kind %q", a.Kind)
	}
}
