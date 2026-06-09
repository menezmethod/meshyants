package ops_test

import (
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/ops"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestPreflightUpdate_OK(t *testing.T) {
	t.Parallel()
	m := &meshyantsv1.UpdateManifest{
		SchemaVersion:  1,
		ReleaseId:      "1.0.1",
		TrustDomain:    "td",
		Targets:        []string{"executor-basic"},
		CompatWindow:   ">=1.0.0",
		Hashes:         map[string][]byte{"bin": {1, 2}},
		RollbackTarget: "1.0.0",
		Issuer:         "queen",
		ValidUntil:     timestamppb.New(time.Now().Add(time.Hour)),
		Signature:      []byte{1},
	}
	r := ops.PreflightUpdate(m, "executor-basic", 32*1024*1024)
	require.True(t, r.OK)
	require.Equal(t, "1.0.0", r.RollbackTarget)
}

func TestPreflightUpdate_LowDisk(t *testing.T) {
	t.Parallel()
	m := &meshyantsv1.UpdateManifest{
		SchemaVersion:  1,
		ReleaseId:      "1.0.1",
		TrustDomain:    "td",
		Targets:        []string{"*"},
		CompatWindow:   ">=1.0.0",
		Hashes:         map[string][]byte{"bin": {1}},
		RollbackTarget: "1.0.0",
		Issuer:         "queen",
		ValidUntil:     timestamppb.New(time.Now().Add(time.Hour)),
		Signature:      []byte{1},
	}
	r := ops.PreflightUpdate(m, "relay", 1024)
	require.False(t, r.OK)
}

func TestValidateConfigDryRun(t *testing.T) {
	t.Parallel()
	require.Error(t, ops.ValidateConfigDryRun([]byte("   ")))
	require.NoError(t, ops.ValidateConfigDryRun([]byte("version: 1\n")))
}
