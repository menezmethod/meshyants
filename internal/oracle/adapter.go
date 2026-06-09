// Package oracle implements the Oracle Interface Agent per docs/v1/03-protocols-and-execution.md.
// It translates human goals into signed TaskAtoms and reads swarm pheromones back to natural language.
package oracle

import (
	"context"
	"errors"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/oracle/policy"
)

// Adapter translates between human language and structured blackboard records.
type Adapter interface {
	// Canonicalize converts a natural-language goal into a validated TaskIntentHeader.
	// If the goal cannot be safely compressed within the DTN budget, it returns
	// header with ContentRef set and ContentOverflow=true (overflow path).
	// If the goal is dangerous or unresolvable, it returns ErrUnresolvable.
	Canonicalize(ctx context.Context, goal string) (*policy.TaskIntentHeader, error)

	// Summarize reads a set of PheromoneRecords and returns a human-readable digest.
	Summarize(ctx context.Context, records []*meshyantsv1.PheromoneRecord) (string, error)
}

var (
	// ErrUnresolvable is returned when a goal cannot be safely translated
	// into a structured TaskAtom (dangerous, ambiguous, or exceeds overflow path).
	ErrUnresolvable = errors.New("oracle: goal unresolvable")
	// ErrOverflow is returned when a goal exceeds the DTN budget and
	// the overflow path is not available.
	ErrOverflow = errors.New("oracle: goal exceeds DTN budget")
)

// Config holds shared adapter configuration.
type Config struct {
	Model          string        // e.g. "gpt-4o", "claude-sonnet-4-20250514"
	RequestTimeout time.Duration // default 30s
	MaxTokens      int
	APIKey         string        // overrides env var
}
