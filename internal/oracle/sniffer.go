// Package oracle provides LLM-backed Oracle Interface Agents.
package oracle

import (
	"context"
	"log"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/oracleview"
)

// Sniffer periodically reads recent pheromones and produces human-readable summaries.
type Sniffer struct {
	Adapter Adapter
	Store   PheromoneStore
	Interval time.Duration
}

// PheromoneStore returns recent pheromone records for a trust domain.
type PheromoneStore interface {
	Recent(ctx context.Context, trustDomain string, limit int) ([]*meshyantsv1.PheromoneRecord, error)
}

// DefaultSnifferInterval is the default period between sniff/summarize cycles.
const DefaultSnifferInterval = 60 * time.Second

// Run starts the periodic sniff loop. It blocks until ctx is cancelled.
// Logs summary on each cycle; errors are logged but do not stop the loop.
func (sn *Sniffer) Run(ctx context.Context) error {
	interval := sn.Interval
	if interval == 0 {
		interval = DefaultSnifferInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Do an initial sniff immediately.
	sn.sniff(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			sn.sniff(ctx)
		}
	}
}

func (sn *Sniffer) sniff(ctx context.Context) {
	if sn.Store == nil {
		return
	}
	records, err := sn.Store.Recent(ctx, "default", 50)
	if err != nil {
		log.Printf("[sniffer] fetch records: %v", err)
		return
	}
	if len(records) == 0 {
		return
	}
	summary, err := sn.Adapter.Summarize(ctx, records)
	if err != nil {
		log.Printf("[sniffer] summarize: %v", err)
		return
	}
	log.Printf("[sniffer] swarm summary: %s", summary)
}

// FormatViews returns oracleview RecordViews for a set of pheromone records.
func FormatViews(records []*meshyantsv1.PheromoneRecord) ([]oracleview.RecordView, error) {
	return oracleview.FormatPheromones(records)
}
