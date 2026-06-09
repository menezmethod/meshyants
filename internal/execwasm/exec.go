// Package execwasm runs ephemeral Wasm tasks with quotas (docs/v1/03-protocols-and-execution.md).
package execwasm

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// ErrTimeout is returned when the Wasm task exceeds its time budget.
var ErrTimeout = errors.New("execwasm: execution budget exceeded")

// ErrMemoryLimit is returned when the Wasm task exceeds its memory quota.
var ErrMemoryLimit = errors.New("execwasm: memory limit exceeded")

// Quotas bound sandbox growth.
type Quotas struct {
	MaxDuration    time.Duration
	MaxMemoryBytes uint32 // bytes; 0 = no limit enforced via wazero
}

// DefaultQuotas returns conservative limits for tests.
func DefaultQuotas() Quotas {
	return Quotas{MaxDuration: 50 * time.Millisecond}
}

// Run executes Wasm with WASI, instantiates once, invokes _start if present, then tears down the runtime.
// Duration and memory quotas are enforced. On quota exhaustion a signal error is returned.
func Run(ctx context.Context, wasm []byte, quotas Quotas) error {
	if quotas.MaxDuration == 0 {
		quotas = DefaultQuotas()
	}
	ctx, cancel := context.WithTimeout(ctx, quotas.MaxDuration)
	defer cancel()

	rCfg := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	if quotas.MaxMemoryBytes > 0 {
		rCfg = rCfg.WithMemoryLimitPages(quotas.MaxMemoryBytes / 65536)
	}

	r := wazero.NewRuntimeWithConfig(ctx, rCfg)
	defer r.Close(ctx)

	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		return err
	}
	compiled, err := r.CompileModule(ctx, wasm)
	if err != nil {
		return err
	}
	modCfg := wazero.NewModuleConfig()
	// Note: memory limit is enforced at RuntimeConfig level; module-level
	// limit is set there. The modCfg is still passed but wazero uses the
	// runtime-level ceiling.
	mod, err := r.InstantiateModule(ctx, compiled, modCfg)
	if err != nil {
		return classifyError(ctx, err)
	}
	defer mod.Close(ctx)

	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return nil
	}
	_, err = fn.Call(ctx)
	if err != nil {
		return classifyError(ctx, err)
	}
	return nil
}

func classifyError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	low := strings.ToLower(err.Error())
	if ctx.Err() != nil || strings.Contains(low, "deadline exceeded") {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}
	if strings.Contains(low, "memory") {
		return fmt.Errorf("%w: %v", ErrMemoryLimit, err)
	}
	return err
}

//go:embed testdata/wasi_arg.wasm
var validWasmBytes []byte

// ValidWasm returns a known-valid WASM module (wasi_arg.wasm).
func ValidWasm() []byte {
	return validWasmBytes
}
