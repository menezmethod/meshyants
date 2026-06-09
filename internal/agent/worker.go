// Package agent implements the field-node worker that pull-consumes TaskAtoms
// from the blackboard and emits pheromones on completion (docs/v1/03-protocols-and-execution.md).
package agent

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/execwasm"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/meshyants/meshyants/v1/internal/ledger/publish"
	"github.com/meshyants/meshyants/v1/internal/oracle/policy"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrUnverifiable  = errors.New("agent: signature verification failed")
	ErrEffectAlready = errors.New("agent: task already executed (effect recorded)")
)

// Executor executes a TaskAtom payload and returns a pheromone kind.
type Executor interface {
	Execute(ctx context.Context, task *meshyantsv1.TaskAtom) (pheromoneKind meshyantsv1.PheromoneKind, detail string)
}

// WasmExecutor delegates to execwasm for .wasm payloads and returns DANGER on error.
type WasmExecutor struct {
	Quotas execwasm.Quotas
}

func (e WasmExecutor) Execute(ctx context.Context, task *meshyantsv1.TaskAtom) (meshyantsv1.PheromoneKind, string) {
	if task == nil || len(task.Payload) == 0 {
		return meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, "empty payload"
	}
	if err := execwasm.Run(ctx, task.Payload, e.Quotas); err != nil {
		return meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, err.Error()
	}
	return meshyantsv1.PheromoneKind_PHEROMONE_KIND_SAFE, "executed"
}

// NativeExecutor acknowledges non-Wasm payloads as SAFE.
type NativeExecutor struct{}

func (e NativeExecutor) Execute(ctx context.Context, task *meshyantsv1.TaskAtom) (meshyantsv1.PheromoneKind, string) {
	if task == nil || len(task.Payload) == 0 {
		return meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, "empty payload"
	}
	_ = ctx
	return meshyantsv1.PheromoneKind_PHEROMONE_KIND_SAFE, "acknowledged"
}

// Worker pull-consumes TaskAtoms from the blackboard and emits pheromones.
type Worker struct {
	Transport     *fabric.JetStreamTransport
	PublishLedger *publish.Ledger // may be nil (async publish wiring optional)
	EffectLedger  *effect.Ledger
	Executor      Executor

	// Identity
	PublicKey   ed25519.PublicKey
	PrivateKey  ed25519.PrivateKey
	Issuer      string
	TrustDomain string

	// Subscription config
	Subject string
	Durable string

	// Verbose logs task requirements, intent JSON payload, and a JSON line for each executor result.
	Verbose bool
}

// Run starts the pull consumer loop. It blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	if w.Executor == nil {
		w.Executor = WasmExecutor{Quotas: execwasm.DefaultQuotas()}
	}

	log.Printf("[worker] starting pull consumer on subject=%q durable=%q", w.Subject, w.Durable)

	if err := w.Transport.PullSubscribe(ctx, w.Subject, w.Durable, func(hctx context.Context, msg *nats.Msg) error {
		return w.handleMessage(hctx, msg)
	}); err != nil {
		return err
	}
	defer w.Transport.Unsubscribe()

	<-ctx.Done()
	return ctx.Err()
}

func (w *Worker) handleMessage(ctx context.Context, msg *nats.Msg) error {
	var task meshyantsv1.TaskAtom
	if err := proto.Unmarshal(msg.Data, &task); err != nil {
		log.Printf("[worker] unmarshal error: %v", err)
		return err
	}

	// 1. Schema version check.
	if task.SchemaVersion == 0 || task.SchemaVersion > 1 {
		log.Printf("[worker] unsupported schema version: %d", task.SchemaVersion)
		d := "unsupported schema version"
		w.logVerboseOutcome(&task, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, d)
		return w.emitPheromone(ctx, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, task.TaskId, d)
	}

	// 2. Re-verify signature using issuer's public key from task.
	issuerPub, err := w.issuerKey(task.Issuer)
	if err != nil {
		log.Printf("[worker] no public key for issuer %q: %v", task.Issuer, err)
		d := "unknown issuer"
		w.logVerboseOutcome(&task, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, d)
		return w.emitPheromone(ctx, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, task.TaskId, d)
	}
	if err := signing.Verify(&task, issuerPub); err != nil {
		log.Printf("[worker] signature verification failed for task %q: %v", task.TaskId, err)
		d := "signature verification failed"
		w.logVerboseOutcome(&task, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, d)
		return w.emitPheromone(ctx, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, task.TaskId, d)
	}

	// 3. Idempotency check via effect ledger.
	recorded, err := w.EffectLedger.TryApply(task.IdempotencyKey)
	if err != nil {
		log.Printf("[worker] effect ledger error: %v", err)
		return err
	}
	if !recorded {
		log.Printf("[worker] task %q already executed (idem=%q), skipping",
			task.TaskId, task.IdempotencyKey)
		if w.Verbose {
			b, _ := json.Marshal(map[string]any{
				"task_id": task.TaskId, "note": "duplicate idempotency_key, no pheromone emitted",
			})
			log.Printf("[worker] result %s", string(b))
		}
		return nil
	}

	// 4. Policy safety check on the requirements/payload.
	if len(task.Payload) > 0 {
		if err := policy.VerifyTaskSafety(task.RequirementsJson, task.Payload); err != nil {
			log.Printf("[worker] unsafe task %q: %v", task.TaskId, err)
			d := err.Error()
			w.logVerboseOutcome(&task, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, d)
			return w.emitPheromone(ctx, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, task.TaskId, d)
		}
	}

	// 5. Execute.
	pheroKind, detail := w.Executor.Execute(ctx, &task)
	log.Printf("[worker] executed task %q → %s: %s", task.TaskId, pheroKind.String(), detail)
	w.logVerboseOutcome(&task, pheroKind, detail)

	// 6. Emit pheromone.
	return w.emitPheromone(ctx, pheroKind, task.TaskId, detail)
}

func (w *Worker) logVerboseOutcome(task *meshyantsv1.TaskAtom, kind meshyantsv1.PheromoneKind, outcome string) {
	if !w.Verbose {
		return
	}
	b, _ := json.Marshal(map[string]any{
		"task_id":               task.TaskId,
		"idempotency_key":       task.IdempotencyKey,
		"issuer":                task.Issuer,
		"pheromone_kind":        kind.String(),
		"executor_or_outcome":   outcome,
		"requirements_json":     task.RequirementsJson,
		"intent_payload_json":     string(task.Payload),
	})
	log.Printf("[worker] result %s", string(b))
}

// issuerKey returns the public key for the given issuer name.
// In production this would query a key registry; for now we use our own key
// when the issuer matches our identity.
func (w *Worker) issuerKey(issuer string) (ed25519.PublicKey, error) {
	if issuer == w.Issuer {
		return w.PublicKey, nil
	}
	// TODO: query key registry (docs/v1/04)
	return nil, fmt.Errorf("unknown issuer: %s", issuer)
}

func (w *Worker) emitPheromone(ctx context.Context, kind meshyantsv1.PheromoneKind, subject, detail string) error {
	rec := &meshyantsv1.PheromoneRecord{
		SchemaVersion: 1,
		RecordId:       fmt.Sprintf("phero-%d-%s", time.Now().UnixNano(), subject),
		Kind:          kind,
		Subject:       subject,
		Strength:      1,
		Issuer:        w.Issuer,
		TrustDomain:   w.TrustDomain,
		CausalParents: []string{subject},
		IssuedAt:      timestamppb.New(time.Now().UTC()),
	}
	if err := signing.Sign(rec, w.PrivateKey); err != nil {
		return fmt.Errorf("sign pheromone: %w", err)
	}
	payload, err := proto.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal pheromone: %w", err)
	}
	subjectPhero := fmt.Sprintf("mesh.phero.%s", w.TrustDomain)
	if err := w.Transport.Publish(ctx, subjectPhero, payload, fabric.PublishOpts{
		MsgID: fabric.StableMsgID(rec.RecordId),
	}); err != nil {
		return fmt.Errorf("publish pheromone: %w", err)
	}
	return nil
}
