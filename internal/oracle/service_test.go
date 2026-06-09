package oracle

import (
	"context"
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/oracle/policy"
	"github.com/stretchr/testify/require"
)

func TestMockAdapter_Canonicalize(t *testing.T) {
	ctx := context.Background()
	adapter := &MockAdapter{
		Header: &policy.TaskIntentHeader{
			Scope:             "deploy the app to staging",
			ProhibitedActions: []string{"rm -rf", "format_disk"},
			DeadlineRFC3339:   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			TransportClass:    "fast-wide",
			CanonicalGoal:     "Deploy app to staging environment",
		},
	}
	header, err := adapter.Canonicalize(ctx, "deploy app to staging")
	require.NoError(t, err)
	require.Equal(t, "deploy the app to staging", header.Scope)
	require.Contains(t, header.ProhibitedActions, "rm -rf")
}

func TestMockAdapter_CanonicalizeError(t *testing.T) {
	ctx := context.Background()
	adapter := &MockAdapter{Err: ErrUnresolvable}
	_, err := adapter.Canonicalize(ctx, "do something dangerous")
	require.ErrorIs(t, err, ErrUnresolvable)
}

func TestMockAdapter_Summarize(t *testing.T) {
	ctx := context.Background()
	adapter := &MockAdapter{}
	records := []*meshyantsv1.PheromoneRecord{
		{Kind: meshyantsv1.PheromoneKind_PHEROMONE_KIND_SAFE, Subject: "task-1"},
		{Kind: meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, Subject: "task-2"},
	}
	summary, err := adapter.Summarize(ctx, records)
	require.NoError(t, err)
	require.Contains(t, summary, "2 records")
}
