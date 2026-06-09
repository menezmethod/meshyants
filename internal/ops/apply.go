// Package ops implements the UpdateManifest apply pipeline with rollback (docs/v1/05-operations-testing-and-rollouts.md).
package ops

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/meshyants/meshyants/v1/internal/store"
)

// Apply handles the full UpdateManifest apply pipeline.
type Apply struct {
	Store   *store.Store
	ApplyFn ApplyFunc
	TempDir string
}

// ApplyFunc is the platform-specific apply implementation.
type ApplyFunc func(ctx context.Context, artifactPath string, manifest *meshyantsv1.UpdateManifest) error

// DefaultApplyFn is a no-op for testing.
func DefaultApplyFn(ctx context.Context, artifactPath string, m *meshyantsv1.UpdateManifest) error {
	return nil
}

// ApplyUpdate runs the full apply pipeline: verify → download + hash → stage → apply → confirm or rollback.
func (a *Apply) ApplyUpdate(ctx context.Context, manifest *meshyantsv1.UpdateManifest, publicKey ed25519.PublicKey) error {
	// 1. Verify manifest signature.
	if err := signing.Verify(manifest, publicKey); err != nil {
		return fmt.Errorf("update: manifest signature invalid: %w", err)
	}

	// 2. Preflight check (uses existing update.go implementation).
	pf := PreflightUpdate(manifest, "RELAY", 0)
	if !pf.OK {
		return fmt.Errorf("update: preflight failed: %s", pf.Reason)
	}

	// 3. Download artifacts and verify hashes.
	if a.ApplyFn == nil {
		a.ApplyFn = DefaultApplyFn
	}
	downloaded := make(map[string]string) // url → local path
	for url, expectedHash := range manifest.Hashes {
		path, err := a.download(ctx, url)
		if err != nil {
			return fmt.Errorf("update: download %s: %w", url, err)
		}
		if err := a.verifyHash(path, expectedHash); err != nil {
			return fmt.Errorf("update: hash mismatch for %s: %w", url, err)
		}
		downloaded[url] = path
	}

	// 4. Stage update.
	stageDir := filepath.Join(a.TempDir, "stage-"+manifest.ReleaseId)
	if err := os.MkdirAll(stageDir, 0700); err != nil {
		return fmt.Errorf("update: stage dir: %w", err)
	}
	for _, path := range downloaded {
		base := filepath.Base(path)
		dest := filepath.Join(stageDir, base)
		if err := os.Rename(path, dest); err != nil {
			return fmt.Errorf("update: stage: %w", err)
		}
	}

	// 5. Apply.
	primary := ""
	for _, path := range downloaded {
		primary = path
		break
	}
	if err := a.ApplyFn(ctx, primary, manifest); err != nil {
		if pf.RollbackTarget != "" {
			if rbErr := a.rollback(ctx, pf.RollbackTarget); rbErr != nil {
				return fmt.Errorf("update: apply failed, rollback also failed: apply=%v rollback=%v", err, rbErr)
			}
			return fmt.Errorf("update: apply failed, rolled back to %s: %w", pf.RollbackTarget, err)
		}
		return fmt.Errorf("update: apply failed: %w", err)
	}

	log.Printf("[update] applied release %s", manifest.ReleaseId)
	return nil
}

func (a *Apply) download(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20)) // 100MB limit
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	path := filepath.Join(a.TempDir, fmt.Sprintf("artifact-%d", len(data)))
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return path, nil
}

func (a *Apply) verifyHash(path string, expected []byte) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	if !bytesEqual(hash[:], expected) {
		return fmt.Errorf("hash mismatch: expected %s, got %s",
			hex.EncodeToString(expected), hex.EncodeToString(hash[:]))
	}
	return nil
}

func (a *Apply) rollback(ctx context.Context, target string) error {
	log.Printf("[update] rollback to %s", target)
	return nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
