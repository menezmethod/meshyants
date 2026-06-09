// Package policy validates structured human intent before TaskAtom publication (audit: U1, U2, U8, U9).
package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMissingField           = errors.New("oracle/policy: missing required safety field")
	ErrInvalidDeadlineRFC3339 = errors.New("oracle/policy: deadline_rfc3339 must be empty or valid RFC3339")
	ErrOversize               = errors.New("oracle/policy: intent exceeds DTN budget; use content_ref or reject")
	ErrDangerousAction        = errors.New("oracle/policy: dangerous action rejected")
)

// TaskIntentHeader is the schema-constrained extraction output before signing (docs/v1/03-protocols-and-execution.md).
type TaskIntentHeader struct {
	Scope              string   `json:"scope"`
	ProhibitedActions  []string `json:"prohibited_actions"`
	DeadlineRFC3339    string   `json:"deadline_rfc3339"`
	TransportClass     string   `json:"transport_class"`
	RequiredApprovals  []string `json:"required_approvals,omitempty"`
	CanonicalGoal      string   `json:"canonical_goal"`
}

// ValidateRequiredSafetyFields returns ErrMissingField if any required field is absent (audit: U1).
// deadline_rfc3339 may be empty when the goal has no deadline; if set, it must parse as time.RFC3339.
func ValidateRequiredSafetyFields(h *TaskIntentHeader) error {
	if h == nil {
		return ErrMissingField
	}
	if strings.TrimSpace(h.Scope) == "" {
		return fmt.Errorf("%w: scope", ErrMissingField)
	}
	if h.ProhibitedActions == nil {
		return fmt.Errorf("%w: prohibited_actions", ErrMissingField)
	}
	// Empty string means no deadline (matches oracle prompt + JSON schema).
	if d := strings.TrimSpace(h.DeadlineRFC3339); d != "" {
		if _, err := time.Parse(time.RFC3339, d); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidDeadlineRFC3339, err)
		}
	}
	if strings.TrimSpace(h.TransportClass) == "" {
		return fmt.Errorf("%w: transport_class", ErrMissingField)
	}
	if strings.TrimSpace(h.CanonicalGoal) == "" {
		return fmt.Errorf("%w: canonical_goal", ErrMissingField)
	}
	return nil
}

// CanonicalJSON produces a stable serialization for idempotency derivation.
func CanonicalJSON(h *TaskIntentHeader) ([]byte, error) {
	if err := ValidateRequiredSafetyFields(h); err != nil {
		return nil, err
	}
	return json.Marshal(h)
}

// DeriveIdempotencyKey is stable for the same canonical goal + client request id (audit: U3).
func DeriveIdempotencyKey(canonicalGoal, clientRequestID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(canonicalGoal) + "\x00" + strings.TrimSpace(clientRequestID)))
	return hex.EncodeToString(sum[:])
}

const DTNIntentBudget = 2048

// CheckDTNSize fails if canonical JSON exceeds budget unless allowOverflow is true (audit: U2).
func CheckDTNSize(h *TaskIntentHeader, allowOverflow bool) ([]byte, error) {
	b, err := CanonicalJSON(h)
	if err != nil {
		return nil, err
	}
	if len(b) <= DTNIntentBudget {
		return b, nil
	}
	if allowOverflow {
		return b, nil
	}
	return nil, fmt.Errorf("%w (%d bytes)", ErrOversize, len(b))
}

var blockedVerbs = []string{"format_disk", "rm_rf", "disable_firewall", "curl_pipe_bash"}

// VerifyTaskSafety rejects known-dangerous patterns in executable hints (audit: U8).
func VerifyTaskSafety(requirementsJSON string, payload []byte) error {
	lower := strings.ToLower(requirementsJSON + "\n" + string(payload))
	for _, b := range blockedVerbs {
		if strings.Contains(lower, b) {
			return fmt.Errorf("%w: %q", ErrDangerousAction, b)
		}
	}
	return nil
}

// EnforceDeltaCaps rejects pathological merge or intent payloads (audit: U9).
func EnforceDeltaCaps(payload []byte, maxBytes, maxDepth int) error {
	if len(payload) > maxBytes {
		return fmt.Errorf("oracle/policy: payload over max bytes %d", maxBytes)
	}
	if maxDepth > 0 {
		var v any
		if err := json.Unmarshal(payload, &v); err == nil {
			if depthAny(v) > maxDepth {
				return fmt.Errorf("oracle/policy: JSON depth exceeded")
			}
		}
	}
	return nil
}

func depthAny(v any) int {
	switch t := v.(type) {
	case map[string]any:
		max := 1
		for _, w := range t {
			d := 1 + depthAny(w)
			if d > max {
				max = d
			}
		}
		return max
	case []any:
		max := 1
		for _, w := range t {
			d := 1 + depthAny(w)
			if d > max {
				max = d
			}
		}
		return max
	default:
		return 1
	}
}
