//go:build integration

package publish_test

import (
	"path/filepath"
	"testing"

	"github.com/meshyants/meshyants/v1/internal/ledger/publish"
	"github.com/stretchr/testify/require"
)

// audit: I3 — ledger survives reopen (crash window recovery).
func TestPublishLedger_I3_Reopen(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "l.db")
	l, err := publish.Open(path)
	require.NoError(t, err)
	require.NoError(t, l.PutPending("p1", "mid", []byte{1}))
	require.NoError(t, l.Close())

	l2, err := publish.Open(path)
	require.NoError(t, err)
	defer l2.Close()
	ent, err := l2.Get("p1")
	require.NoError(t, err)
	require.Equal(t, publish.StatePending, ent.State)
}
