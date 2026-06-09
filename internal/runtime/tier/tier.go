// Package tier implements runtime tier detection per docs/v1/00-overview.md.
package tier

import (
	"context"
	"runtime"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/tetratelabs/wazero"
)

// Options tune tier detection.
type Options struct {
	// MinMemoryBytesForWasm is the host memory heuristic for allowing executor-wasm (default 256MiB).
	MinMemoryBytesForWasm uint64
}

// DefaultOptions returns conservative defaults.
func DefaultOptions() Options {
	return Options{MinMemoryBytesForWasm: 256 * 1024 * 1024}
}

// Detect returns the highest runtime tier this process likely supports.
func Detect(ctx context.Context, opt Options) meshyantsv1.RuntimeTier {
	_ = ctx
	// All nodes can relay at minimum.
	tier := meshyantsv1.RuntimeTier_RUNTIME_TIER_RELAY
	// Native executor: assume general-purpose OS (Linux/macOS/Windows class).
	switch runtime.GOOS {
	case "linux", "darwin", "windows", "freebsd":
		tier = meshyantsv1.RuntimeTier_RUNTIME_TIER_EXECUTOR_BASIC
	default:
		return tier
	}
	if opt.MinMemoryBytesForWasm == 0 {
		opt = DefaultOptions()
	}
	if sysMem := readSysTotalMemory(); sysMem > 0 && sysMem < opt.MinMemoryBytesForWasm {
		return tier
	}
	// Wasm tier: wazero runtime can be constructed on this host.
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	tier = meshyantsv1.RuntimeTier_RUNTIME_TIER_EXECUTOR_WASM
	return tier
}
