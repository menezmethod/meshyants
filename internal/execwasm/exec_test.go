package execwasm

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/wasi_arg.wasm
var wasiArgWasm []byte

//go:embed testdata/infinite_loop.wasm
var infiniteLoopWasm []byte

func TestRun_WASIArg(t *testing.T) {
	ctx := context.Background()
	err := Run(ctx, wasiArgWasm, Quotas{MaxDuration: 2 * time.Second})
	require.NoError(t, err)
}

func TestRun_ValidWasm(t *testing.T) {
	ctx := context.Background()
	err := Run(ctx, ValidWasm(), DefaultQuotas())
	require.NoError(t, err)
}

func TestRun_MemoryLimit(t *testing.T) {
	ctx := context.Background()
	// 512KB memory limit — ValidWasm is tiny, should pass.
	err := Run(ctx, ValidWasm(), Quotas{MaxDuration: 2 * time.Second, MaxMemoryBytes: 512 * 1024})
	require.NoError(t, err)
}

func TestRun_InfiniteLoopTimeout(t *testing.T) {
	ctx := context.Background()
	err := Run(ctx, infiniteLoopWasm, Quotas{MaxDuration: 30 * time.Millisecond})
	require.Error(t, err)
	require.True(t,
		errors.Is(err, ErrTimeout) || strings.Contains(strings.ToLower(err.Error()), "deadline exceeded"),
		"got %v", err,
	)
}

func TestRun_Timeout(t *testing.T) {
	ctx := context.Background()
	// Infinite loop with very short timeout.
	err := Run(ctx, infiniteLoopWasm, Quotas{MaxDuration: 1 * time.Millisecond})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrTimeout) || strings.Contains(err.Error(), "deadline exceeded"))
}
