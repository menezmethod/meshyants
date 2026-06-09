//go:build integration

package effect_test

import (
	"path/filepath"
	"testing"

	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/stretchr/testify/require"
)

// audit: I4 — effect ledger survives reopen (consumer crash window).
func TestEffectLedger_I4_Reopen(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "e.db")
	l, err := effect.Open(path)
	require.NoError(t, err)
	applied, err := l.TryApply("fx1")
	require.NoError(t, err)
	require.True(t, applied)
	require.NoError(t, l.Close())

	l2, err := effect.Open(path)
	require.NoError(t, err)
	defer l2.Close()
	applied, err = l2.TryApply("fx1")
	require.NoError(t, err)
	require.False(t, applied)
}
