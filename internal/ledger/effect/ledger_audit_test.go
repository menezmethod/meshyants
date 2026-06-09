package effect_test

import (
	"path/filepath"
	"testing"

	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/stretchr/testify/require"
)

// audit: U5 — effect ledger suppresses repeated side effects.
func TestEffectLedger_U5_Idempotent(t *testing.T) {
	t.Parallel()
	l, err := effect.Open(filepath.Join(t.TempDir(), "e.db"))
	require.NoError(t, err)
	defer l.Close()

	applied, err := l.TryApply("idem-1")
	require.NoError(t, err)
	require.True(t, applied)

	applied, err = l.TryApply("idem-1")
	require.NoError(t, err)
	require.False(t, applied)

	ok, err := l.Has("idem-1")
	require.NoError(t, err)
	require.True(t, ok)
}
