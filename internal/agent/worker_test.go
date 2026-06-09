//go:build e2e

package agent_test

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/agent"
	"github.com/meshyants/meshyants/v1/internal/execwasm"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/meshyants/meshyants/v1/test/natstest"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestWorker_TaskRoundTrip tests the full worker loop: publish task → consume → execute → pheromone emitted.
func TestWorker_TaskRoundTrip(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	_, js, tr := setupNATSAndTransport(ctx, t)

	pubLedger, err := effect.Open(filepath.Join(tmp, "effect.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = pubLedger.Close() })

	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	taskSubject := fmt.Sprintf("mesh.task.%d", time.Now().UnixNano())
	durable := fmt.Sprintf("worker-%d", time.Now().UnixNano())

	task := &meshyantsv1.TaskAtom{
		SchemaVersion:    1,
		TaskId:           "task-wasm-1",
		IdempotencyKey:   "idem-wasm-1",
		Issuer:           "test-issuer",
		TrustDomain:      "td-test",
		CausalParents:    []string{"genesis"},
		RequirementsJson: `{"transport":"fast-local"}`,
		Payload:          execwasm.ValidWasm(),
		ExpiresAt:        timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(task, priv))

	payload, err := proto.Marshal(task)
	require.NoError(t, err)
	_, err = js.Publish(taskSubject, payload, nats.MsgId(task.IdempotencyKey))
	require.NoError(t, err)

	w := &agent.Worker{
		Transport:    tr,
		EffectLedger: pubLedger,
		Executor:     agent.WasmExecutor{Quotas: execwasm.DefaultQuotas()},
		PublicKey:    pub,
		PrivateKey:   priv,
		Issuer:       "test-issuer",
		TrustDomain:  "td-test",
		Subject:      taskSubject,
		Durable:      durable,
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	t.Cleanup(cancelRun)

	errCh := make(chan error, 1)
	go func() { errCh <- w.Run(runCtx) }()

	pheroSubj := fmt.Sprintf("mesh.phero.%s", w.TrustDomain)
	sub, err := js.PullSubscribe(pheroSubj, fmt.Sprintf("phero-check-%d", time.Now().UnixNano()), nats.BindStream("MESHYANTS"), nats.DeliverAll())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	var fetched []*nats.Msg
	deadline := time.Now().Add(15 * time.Second)
	for len(fetched) < 1 && time.Now().Before(deadline) {
		batch, err := sub.Fetch(1, nats.MaxWait(time.Second))
		if err != nil && err != nats.ErrTimeout {
			require.NoError(t, err)
		}
		fetched = append(fetched, batch...)
	}
	require.Len(t, fetched, 1)

	var rec meshyantsv1.PheromoneRecord
	require.NoError(t, proto.Unmarshal(fetched[0].Data, &rec))
	require.Equal(t, task.TaskId, rec.Subject)
	require.Equal(t, meshyantsv1.PheromoneKind_PHEROMONE_KIND_SAFE, rec.Kind)

	cancelRun()
	err = <-errCh
	require.ErrorIs(t, err, context.Canceled)
}

// TestWorker_Idempotency verifies replay of the same idempotency key suppresses duplicate execution.
func TestWorker_Idempotency(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	_, js, tr := setupNATSAndTransport(ctx, t)

	pubLedger, err := effect.Open(filepath.Join(tmp, "effect_idem.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = pubLedger.Close() })

	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	taskSubject := fmt.Sprintf("mesh.task.idem-%d", time.Now().UnixNano())
	durable := fmt.Sprintf("worker-idem-%d", time.Now().UnixNano())

	task1 := validSignedTask(t, priv, "task-id-1", "idem-shared")
	task2 := validSignedTask(t, priv, "task-id-2", "idem-shared")

	for _, task := range []*meshyantsv1.TaskAtom{task1, task2} {
		payload, err := proto.Marshal(task)
		require.NoError(t, err)
		_, err = js.Publish(taskSubject, payload, nats.MsgId(task.IdempotencyKey))
		require.NoError(t, err)
	}

	w := &agent.Worker{
		Transport:    tr,
		EffectLedger: pubLedger,
		Executor:     agent.NativeExecutor{},
		PublicKey:    pub,
		PrivateKey:   priv,
		Issuer:       "test-issuer",
		TrustDomain:  "td-test",
		Subject:      taskSubject,
		Durable:      durable,
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	t.Cleanup(cancelRun)

	errCh := make(chan error, 1)
	go func() { errCh <- w.Run(runCtx) }()

	require.Eventually(t, func() bool {
		ok, err := pubLedger.Has("idem-shared")
		return err == nil && ok
	}, 15*time.Second, 100*time.Millisecond)

	time.Sleep(400 * time.Millisecond)

	cancelRun()
	err = <-errCh
	require.ErrorIs(t, err, context.Canceled)

	applied, err := pubLedger.TryApply("idem-shared")
	require.NoError(t, err)
	require.False(t, applied)
}

// TestWorker_DangerOnUnsafeTask verifies a dangerous payload gets DANGER pheromone.
func TestWorker_DangerOnUnsafeTask(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	_, js, tr := setupNATSAndTransport(ctx, t)

	pubLedger, err := effect.Open(filepath.Join(tmp, "effect_danger.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = pubLedger.Close() })

	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	taskSubject := fmt.Sprintf("mesh.task.bad-%d", time.Now().UnixNano())
	durable := fmt.Sprintf("worker-bad-%d", time.Now().UnixNano())

	task := validSignedTask(t, priv, "task-danger", "idem-danger")
	task.Payload = []byte(`{"cmd":"rm -rf /"}`)

	payload, err := proto.Marshal(task)
	require.NoError(t, err)
	_, err = js.Publish(taskSubject, payload, nats.MsgId(task.IdempotencyKey))
	require.NoError(t, err)

	w := &agent.Worker{
		Transport:    tr,
		EffectLedger: pubLedger,
		Executor:     agent.NativeExecutor{},
		PublicKey:    pub,
		PrivateKey:   priv,
		Issuer:       "test-issuer",
		TrustDomain:  "td-test",
		Subject:      taskSubject,
		Durable:      durable,
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	t.Cleanup(cancelRun)

	errCh := make(chan error, 1)
	go func() { errCh <- w.Run(runCtx) }()

	pheroSubj := fmt.Sprintf("mesh.phero.%s", w.TrustDomain)
	sub, err := js.PullSubscribe(pheroSubj, fmt.Sprintf("phero-check-danger-%d", time.Now().UnixNano()), nats.BindStream("MESHYANTS"), nats.DeliverAll())
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	var fetched []*nats.Msg
	deadline := time.Now().Add(15 * time.Second)
	for len(fetched) < 1 && time.Now().Before(deadline) {
		batch, err := sub.Fetch(1, nats.MaxWait(time.Second))
		if err != nil && err != nats.ErrTimeout {
			require.NoError(t, err)
		}
		fetched = append(fetched, batch...)
	}
	require.Len(t, fetched, 1)

	var rec meshyantsv1.PheromoneRecord
	require.NoError(t, proto.Unmarshal(fetched[0].Data, &rec))
	require.Equal(t, meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER, rec.Kind)
	require.Equal(t, task.TaskId, rec.Subject)

	cancelRun()
	err = <-errCh
	require.ErrorIs(t, err, context.Canceled)
}

func setupNATSAndTransport(ctx context.Context, t *testing.T) (*nats.Conn, nats.JetStreamContext, *fabric.JetStreamTransport) {
	t.Helper()
	url := natstest.StartJetStreamURL(ctx, t)
	nc, err := natstest.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = nc.Drain() })
	js, err := nc.JetStream()
	require.NoError(t, err)
	tr, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(tr.Close)
	return nc, js, tr
}

func validSignedTask(t *testing.T, priv ed25519.PrivateKey, taskID, idemKey string) *meshyantsv1.TaskAtom {
	t.Helper()
	task := &meshyantsv1.TaskAtom{
		SchemaVersion:    1,
		TaskId:           taskID,
		IdempotencyKey:   idemKey,
		Issuer:           "test-issuer",
		TrustDomain:      "td-test",
		CausalParents:    []string{"genesis"},
		RequirementsJson: `{"transport":"fast-local"}`,
		Payload:          []byte(`{}`),
		ExpiresAt:        timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(task, priv))
	return task
}
