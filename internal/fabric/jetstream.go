package fabric

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// JetStreamConfig names the stream used for MeshyAnts records.
type JetStreamConfig struct {
	StreamName     string
	StreamSubjects []string
}

// DefaultJetStreamConfig returns stream "MESHYANTS" on mesh.> subjects.
func DefaultJetStreamConfig() JetStreamConfig {
	return JetStreamConfig{
		StreamName:     "MESHYANTS",
		StreamSubjects: []string{"mesh.>"},
	}
}

// JetStreamTransport publishes to NATS JetStream with optional dedupe Msg-Id
// and supports pull-based subscription.
type JetStreamTransport struct {
	nc  *nats.Conn
	js  nats.JetStreamContext
	cfg JetStreamConfig

	sub       *nats.Subscription
	pollStop  chan struct{}
	pollDone  chan struct{}
}

// ConnectJetStream connects and ensures the JetStream stream exists.
func ConnectJetStream(ctx context.Context, natsURL string, cfg JetStreamConfig) (*JetStreamTransport, error) {
	var nc *nats.Conn
	var err error
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		deadline = time.Now().Add(60 * time.Second)
	}
	for attempt := 0; attempt < 40; attempt++ {
		nc, err = nats.Connect(natsURL, nats.Timeout(10*time.Second))
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		_ = nc.Drain()
		return nil, err
	}
	if _, err := js.StreamInfo(cfg.StreamName); err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     cfg.StreamName,
			Subjects: cfg.StreamSubjects,
			Storage:  nats.FileStorage,
		})
		if err != nil {
			_ = nc.Drain()
			return nil, fmt.Errorf("fabric: stream %q: %w", cfg.StreamName, err)
		}
	}
	return &JetStreamTransport{nc: nc, js: js, cfg: cfg}, nil
}

// Publish sends data to subject with JetStream deduplication when MsgID is set.
func (t *JetStreamTransport) Publish(ctx context.Context, subject string, data []byte, opts PublishOpts) error {
	_ = ctx
	var pubOpts []nats.PubOpt
	if opts.MsgID != "" {
		pubOpts = append(pubOpts, nats.MsgId(opts.MsgID))
	}
	_, err := t.js.Publish(subject, data, pubOpts...)
	return err
}

// Close drains the underlying connection.
func (t *JetStreamTransport) Close() {
	if t == nil || t.nc == nil {
		return
	}
	if t.sub != nil {
		_ = t.sub.Unsubscribe()
	}
	_ = t.nc.Drain()
}

// StableMsgID derives a broker-safe dedupe id from an idempotency key.
func StableMsgID(idempotencyKey string) string {
	sum := sha256.Sum256([]byte(idempotencyKey))
	return hex.EncodeToString(sum[:16])
}

// PullSubscribe creates a durable pull subscription and drives the handler loop.
func (t *JetStreamTransport) PullSubscribe(ctx context.Context, subject, durable string, handler MsgHandler) error {
	sub, err := t.js.PullSubscribe(subject, durable, nats.BindStream(t.cfg.StreamName))
	if err != nil {
		return fmt.Errorf("fabric: PullSubscribe: %w", err)
	}
	t.sub = sub
	t.pollStop = make(chan struct{})
	t.pollDone = make(chan struct{})

	go t.pollLoop(sub, handler)

	return nil
}

func (t *JetStreamTransport) pollLoop(sub *nats.Subscription, handler MsgHandler) {
	defer close(t.pollDone)
	for {
		select {
		case <-t.pollStop:
			return
		default:
		}
		msgs, err := sub.Fetch(16, nats.MaxWait(5*time.Second))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			return
		}
		for _, msg := range msgs {
			hctx := context.Background()
			if err := handler(hctx, msg); err != nil {
				_ = msg.Nak()
			} else {
				_ = msg.Ack()
			}
		}
	}
}

// Unsubscribe stops the poll loop and the subscription.
func (t *JetStreamTransport) Unsubscribe() {
	if t.pollStop != nil {
		close(t.pollStop)
	}
	if t.pollDone != nil {
		<-t.pollDone
	}
	if t.sub != nil {
		_ = t.sub.Unsubscribe()
		t.sub = nil
	}
}
