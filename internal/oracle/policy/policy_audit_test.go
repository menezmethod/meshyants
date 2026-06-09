package policy_test

import (
	"strings"
	"testing"

	"github.com/meshyants/meshyants/v1/internal/oracle/policy"
	"github.com/stretchr/testify/require"
)

// audit: U1 — required safety fields.
func TestOracle_U1_RequiredFields(t *testing.T) {
	t.Parallel()
	h := &policy.TaskIntentHeader{Scope: "x"}
	require.ErrorIs(t, policy.ValidateRequiredSafetyFields(h), policy.ErrMissingField)
	full := &policy.TaskIntentHeader{
		Scope:             "s",
		ProhibitedActions: []string{},
		DeadlineRFC3339:   "2027-01-01T00:00:00Z",
		TransportClass:    "delay-tolerant",
		CanonicalGoal:     "do thing",
	}
	require.NoError(t, policy.ValidateRequiredSafetyFields(full))

	noDeadline := &policy.TaskIntentHeader{
		Scope:             "s",
		ProhibitedActions: []string{},
		DeadlineRFC3339:   "",
		TransportClass:    "delay-tolerant",
		CanonicalGoal:     "do thing",
	}
	require.NoError(t, policy.ValidateRequiredSafetyFields(noDeadline))

	require.ErrorIs(t, policy.ValidateRequiredSafetyFields(&policy.TaskIntentHeader{
		Scope:             "s",
		ProhibitedActions: []string{},
		DeadlineRFC3339:   "not-a-date",
		TransportClass:    "delay-tolerant",
		CanonicalGoal:     "do thing",
	}), policy.ErrInvalidDeadlineRFC3339)
}

// audit: U2 — oversize without overflow flag.
func TestOracle_U2_Oversize(t *testing.T) {
	t.Parallel()
	h := &policy.TaskIntentHeader{
		Scope:             "s",
		ProhibitedActions: []string{},
		DeadlineRFC3339:   "2027-01-01T00:00:00Z",
		TransportClass:    "delay-tolerant",
		CanonicalGoal:     strings.Repeat("Z", policy.DTNIntentBudget+10),
	}
	_, err := policy.CheckDTNSize(h, false)
	require.ErrorIs(t, err, policy.ErrOversize)
	_, err = policy.CheckDTNSize(h, true)
	require.NoError(t, err)
}

// audit: U3 — stable idempotency key.
func TestOracle_U3_IdempotencyStable(t *testing.T) {
	t.Parallel()
	a := policy.DeriveIdempotencyKey("goal", "req-1")
	b := policy.DeriveIdempotencyKey("goal", "req-1")
	require.Equal(t, a, b)
	require.NotEqual(t, a, policy.DeriveIdempotencyKey("goal2", "req-1"))
}

// audit: U8 — dangerous task rejection.
func TestOracle_U8_Dangerous(t *testing.T) {
	t.Parallel()
	err := policy.VerifyTaskSafety(`{"x":1}`, []byte("please format_disk now"))
	require.ErrorIs(t, err, policy.ErrDangerousAction)
}

// audit: U9 — delta caps.
func TestOracle_U9_DeltaCaps(t *testing.T) {
	t.Parallel()
	require.Error(t, policy.EnforceDeltaCaps([]byte("xx"), 1, 0))
	deep := `{"x":1}`
	for range 12 {
		deep = `{"k":` + deep + `}`
	}
	require.Error(t, policy.EnforceDeltaCaps([]byte(deep), 100000, 4))
	require.NoError(t, policy.EnforceDeltaCaps([]byte(`{"a":1}`), 1000, 20))
}
