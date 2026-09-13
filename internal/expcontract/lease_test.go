package expcontract

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadLeaseRejectsTaskSet(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "docs", "experiments", "examples", "phase-a-synthetic", "tasks.json")
	_, err := loadLease(path)
	if err == nil {
		t.Fatal("expected tasks.json to be rejected as a lease")
	}
}

func TestLoadLeaseAcceptsFenceDoc(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "lease.json")
	if err := os.WriteFile(path, []byte(`{"kind":"exclusive_fence","fencing":"token","exactly_once":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	lease, err := loadLease(path)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Kind != "exclusive_fence" || !lease.ExactlyOnce {
		t.Fatalf("unexpected lease %#v", lease)
	}
}
