// Package fabric abstracts the blackboard transport (docs/v1/01-platform-architecture.md).
package fabric

import (
	"context"

	"github.com/nats-io/nats.go"
)

// TransportProfile names delay and reliability class for tuning.
type TransportProfile string

const (
	FastLocal     TransportProfile = "fast-local"
	FastWide      TransportProfile = "fast-wide"
	DelayTolerant TransportProfile = "delay-tolerant"
)

// PublishOpts carries broker-specific metadata.
type PublishOpts struct {
	MsgID string // JetStream deduplication id (stable from idempotency_key).
}

// Transport is the minimal blackboard publish surface (subscribe can extend later).
type Transport interface {
	Publish(ctx context.Context, subject string, data []byte, opts PublishOpts) error
	Close()
}

// MsgHandler is called for each message delivered by the pull consumer.
// Return nil to ack the message, or return an error to nak (negative ack, redeliver).
type MsgHandler func(ctx context.Context, msg *nats.Msg) error

// PullConsumer is the subscribe side of the blackboard.
type PullConsumer interface {
	// PullSubscribe binds or creates a durable pull subscription on subject.
	// The durable name must be stable across restarts.
	PullSubscribe(ctx context.Context, subject, durable string, handler MsgHandler) error
	// Unsubscribe tears down the subscription.
	Unsubscribe()
}

// Carrier is implemented by the JetStream transport to expose raw JetStreamContext.
type Carrier interface {
	JetStreamContext() nats.JetStreamContext
}

// JetStreamCarrier exposes the underlying JetStream context.
func (t *JetStreamTransport) JetStreamContext() nats.JetStreamContext {
	return t.js
}
