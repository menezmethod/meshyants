// Package merge enforces CRDT merge family boundaries per docs/v1/04-security-resilience-and-consensus.md.
package merge

import (
	"errors"
	"strings"
)

var ErrDisallowedFamily = errors.New("merge: artifact family not allowed for naive CRDT merge")

// Known structured families allowed on delay-tolerant CRDT-style merge paths.
var allowedFamilies = map[string]struct{}{
	"pheromone_summary": {},
	"capability_summary": {},
	"sensor_aggregate": {},
}

// AllowNaiveCRDT returns nil if family may use the naive CRDT merge path (audit: U7).
func AllowNaiveCRDT(artifactFamily string) error {
	f := strings.TrimSpace(strings.ToLower(artifactFamily))
	if f == "" {
		return ErrDisallowedFamily
	}
	if _, ok := allowedFamilies[f]; ok {
		return nil
	}
	// Explicitly reject website/code/config bundle families.
	switch f {
	case "website", "website_bundle", "code_repo", "config_bundle", "binary_artifact":
		return ErrDisallowedFamily
	default:
		return ErrDisallowedFamily
	}
}
