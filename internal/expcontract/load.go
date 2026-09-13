package expcontract

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Load reads a run spec and its referenced task set and worker pool, then validates.
func Load(path string) (*RunSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read run spec: %w", err)
	}
	var spec RunSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("parse run spec: %w", err)
	}
	base := filepath.Dir(path)
	tasks, err := loadTaskSet(filepath.Join(base, spec.Workload.TasksRef))
	if err != nil {
		return nil, err
	}
	pool, err := loadPool(filepath.Join(base, spec.WorkerPoolRef))
	if err != nil {
		return nil, err
	}
	spec.Tasks = tasks
	spec.Pool = pool
	spec.baseDir = base
	if spec.LeaseContractRef != "" {
		if _, err := os.Stat(filepath.Join(base, spec.LeaseContractRef)); err != nil {
			return nil, fmt.Errorf("lease_contract_ref %q: %w", spec.LeaseContractRef, err)
		}
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return &spec, nil
}

func loadTaskSet(path string) (*TaskSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read task set %s: %w", path, err)
	}
	var set TaskSet
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, fmt.Errorf("parse task set %s: %w", path, err)
	}
	return &set, nil
}

func loadPool(path string) (*WorkerPool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read worker pool %s: %w", path, err)
	}
	var pool WorkerPool
	if err := json.Unmarshal(raw, &pool); err != nil {
		return nil, fmt.Errorf("parse worker pool %s: %w", path, err)
	}
	return &pool, nil
}
