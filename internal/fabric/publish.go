// Package fabric provides the blackboard transport and async publish ledger integration.
// Async publish is wired to the publish ledger so that every outbound message is
// durably tracked before it leaves the node (docs/v1/10-failure-oriented-design-audit.md Pattern 2).
package fabric

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/nats-io/nats.go"
)

// PublishResult captures the outcome of an async publish.
type PublishResult struct {
	PublishID string
	MsgID    string
	Err      error
}

// PublishingService provides async publish with ledger tracking.
type PublishingService struct {
	transport *JetStreamTransport
	ledger    LedgerWriter
}

// LedgerWriter abstracts the publish ledger for testability and optional wiring.
type LedgerWriter interface {
	PutPending(publishID, msgID string, payloadHash []byte) error
	MarkAcked(publishID string) error
}

// NewPublishingService wraps a JetStreamTransport with publish ledger integration.
// Pass nil to disable ledger integration.
func NewPublishingService(t *JetStreamTransport, l LedgerWriter) *PublishingService {
	return &PublishingService{transport: t, ledger: l}
}

// AsyncPublish persists publish intent to the ledger, sends asynchronously via JetStream,
// and transitions the ledger entry to Acked on success. It is safe to call concurrently.
func (s *PublishingService) AsyncPublish(ctx context.Context, subject string, data []byte, opts PublishOpts) <-chan PublishResult {
	resultCh := make(chan PublishResult, 1)

	publishID := opts.MsgID
	if publishID == "" {
		publishID = StableMsgID(string(data))
	}
	payloadHash := sha256.Sum256(data)

	// 1. Persist pending BEFORE sending (docs/v1/10 Pattern 2).
	if s.ledger != nil {
		if err := s.ledger.PutPending(publishID, opts.MsgID, payloadHash[:]); err != nil {
			resultCh <- PublishResult{PublishID: publishID, MsgID: opts.MsgID, Err: fmt.Errorf("ledger put pending: %w", err)}
			return resultCh
		}
	}

	// 2. Async publish. PublishAsync returns immediately; acks are delivered to the future.
	paf, err := s.transport.js.PublishAsync(subject, data, nats.MsgId(opts.MsgID))
	if err != nil {
		// Fall back to synchronous publish if the server doesn't support async.
		_, syncErr := s.transport.js.Publish(subject, data, nats.MsgId(opts.MsgID))
		if s.ledger != nil && syncErr == nil {
			_ = s.ledger.MarkAcked(publishID) // best-effort
		}
		resultCh <- PublishResult{PublishID: publishID, MsgID: opts.MsgID, Err: syncErr}
		return resultCh
	}

	// 3. Wait for async ack in background and update ledger.
	go func() {
		select {
		case <-paf.Ok():
			if s.ledger != nil {
				_ = s.ledger.MarkAcked(publishID)
			}
			resultCh <- PublishResult{PublishID: publishID, MsgID: opts.MsgID, Err: nil}
		case err := <-paf.Err():
			resultCh <- PublishResult{PublishID: publishID, MsgID: opts.MsgID, Err: fmt.Errorf("async publish: %w", err)}
		case <-ctx.Done():
			resultCh <- PublishResult{PublishID: publishID, MsgID: opts.MsgID, Err: ctx.Err()}
		}
	}()

	return resultCh
}
