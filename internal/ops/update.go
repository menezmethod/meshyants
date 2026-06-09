// Package ops implements UpdateManifest preflight and rollback metadata handling (docs/v1/05-operations-testing-and-rollouts.md).
package ops

import (
	"errors"
	"fmt"
	"strings"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/validate"
)

var ErrRollbackRequired = errors.New("ops: rollout failed; apply rollback_target")

// PreflightResult is the outcome of local preflight checks.
type PreflightResult struct {
	RollbackTarget string
	ReleaseID      string
	OK             bool
	Reason         string
}

// PreflightUpdate validates manifest and node compatibility hints (dry-run).
func PreflightUpdate(m *meshyantsv1.UpdateManifest, runtimeTier string, freeSpaceBytes uint64) PreflightResult {
	if err := validate.UpdateManifest(m); err != nil {
		return PreflightResult{OK: false, Reason: err.Error(), RollbackTarget: m.GetRollbackTarget()}
	}
	if m.RollbackTarget == "" {
		return PreflightResult{OK: false, Reason: "rollback_target required for safe rollout", RollbackTarget: ""}
	}
	// Minimal tier gate: manifest targets list may name tiers; if non-empty, require membership.
	if len(m.Targets) > 0 {
		match := false
		for _, t := range m.Targets {
			if strings.EqualFold(strings.TrimSpace(t), strings.TrimSpace(runtimeTier)) || t == "*" {
				match = true
				break
			}
		}
		if !match {
			return PreflightResult{
				OK:             false,
				Reason:         fmt.Sprintf("runtime tier %q not in manifest targets", runtimeTier),
				RollbackTarget: m.RollbackTarget,
				ReleaseID:      m.ReleaseId,
			}
		}
	}
	const minFree = uint64(16 * 1024 * 1024)
	if freeSpaceBytes < minFree {
		return PreflightResult{
			OK:             false,
			Reason:         fmt.Sprintf("insufficient free space: %d < %d", freeSpaceBytes, minFree),
			RollbackTarget: m.RollbackTarget,
			ReleaseID:      m.ReleaseId,
		}
	}
	return PreflightResult{OK: true, ReleaseID: m.ReleaseId, RollbackTarget: m.RollbackTarget}
}

// ValidateConfigDryRun parses signed config JSON/YAML in future; placeholder returns nil for empty input.
func ValidateConfigDryRun(configBytes []byte) error {
	if len(strings.TrimSpace(string(configBytes))) == 0 {
		return fmt.Errorf("ops: empty config")
	}
	return nil
}
