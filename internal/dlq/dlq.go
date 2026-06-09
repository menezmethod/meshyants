// Package dlq provides the Dead Letter Queue for unverifiable or invalid blackboard records
// (docs/v1/03-protocols-and-execution.md and docs/v1/10-failure-oriented-design-audit.md).
package dlq

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// DLQ publishes failed records to a JetStream stream for later analysis and retry.
type DLQ struct {
	js      nats.JetStreamContext
	stream  string
	subject string
}

// NewDLQ creates a DLQ backed by the MESHYANTS.DLQ stream.
func NewDLQ(js nats.JetStreamContext) (*DLQ, error) {
	dlq := &DLQ{js: js, stream: "MESHYANTS_DLQ", subject: "mesh.dlq.>"}
	// Ensure the DLQ stream exists.
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     dlq.stream,
		Subjects: []string{dlq.subject},
		Storage:  nats.FileStorage,
	})
	if err != nil {
		// Stream may already exist; that's fine.
		_, _ = js.StreamInfo(dlq.stream)
	}
	return dlq, err
}

// Enqueue publishes a failed record to the DLQ with metadata headers.
func (dlq *DLQ) Enqueue(ctx context.Context, subject string, data []byte, reason string) error {
	headers := nats.Header{}
	headers.Set("X-DLQ-Reason", reason)
	headers.Set("X-DLQ-Time", time.Now().UTC().Format(time.RFC3339))
	headers.Set("X-DLQ-Original-Subject", subject)

	msg := nats.NewMsg("mesh.dlq." + subject)
	msg.Data = data
	msg.Header = headers

	_, err := dlq.js.PublishMsg(msg)
	return err
}

// DLQEntry describes one DLQ entry for inspection.
type DLQEntry struct {
	Subject      string
	Reason       string
	EnqueuedAt   time.Time
	OriginalData []byte
}

// ReadAll retrieves all current DLQ entries (newest first).
func (dlq *DLQ) ReadAll(ctx context.Context) ([]DLQEntry, error) {
	sub, err := dlq.js.PullSubscribe("mesh.dlq.>", "dlq-reader", nats.DeliverAll())
	if err != nil {
		return nil, fmt.Errorf("pull subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	msgs, err := sub.Fetch(100, nats.MaxWait(2*time.Second))
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	entries := make([]DLQEntry, 0, len(msgs))
	for _, msg := range msgs {
		e := DLQEntry{
			Subject:      msg.Header.Get("X-DLQ-Original-Subject"),
			Reason:       msg.Header.Get("X-DLQ-Reason"),
			OriginalData: msg.Data,
		}
		if t, err := time.Parse(time.RFC3339, msg.Header.Get("X-DLQ-Time")); err == nil {
			e.EnqueuedAt = t
		}
		entries = append(entries, e)
	}
	return entries, nil
}
