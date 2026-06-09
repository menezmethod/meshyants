package publish_test

import (
	"path/filepath"
	"testing"

	"github.com/meshyants/meshyants/v1/internal/ledger/publish"
	"github.com/stretchr/testify/require"
)

// audit: U4 — publish ledger state machine is atomic (pending -> acked, no skip).
func TestPublishLedger_U4_StateMachine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "pub.db")
	l, err := publish.Open(path)
	require.NoError(t, err)
	defer l.Close()

	require.NoError(t, l.PutPending("p1", "mid-1", []byte{9, 9}))
	ent, err := l.Get("p1")
	require.NoError(t, err)
	require.Equal(t, publish.StatePending, ent.State)

	require.NoError(t, l.MarkAcked("p1"))
	ent, err = l.Get("p1")
	require.NoError(t, err)
	require.Equal(t, publish.StateAcked, ent.State)

	require.Error(t, l.MarkAcked("p1"))
}

func TestPublishLedger_U4_NoResurrectAcked(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l, err := publish.Open(filepath.Join(dir, "p.db"))
	require.NoError(t, err)
	defer l.Close()

	require.NoError(t, l.PutPending("x", "m", nil))
	require.NoError(t, l.MarkAcked("x"))
	err = l.PutPending("x", "m2", nil)
	require.ErrorIs(t, err, publish.ErrResurrected)
}
