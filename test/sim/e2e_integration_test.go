//go:build e2e

package sim_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/agent"
	"github.com/meshyants/meshyants/v1/internal/execwasm"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/meshyants/meshyants/v1/internal/oracle"
	"github.com/meshyants/meshyants/v1/internal/routing"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/meshyants/meshyants/v1/test/natstest"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestE2E_MultiDeviceMesh tests a 3-node mesh: RPi5 (worker), Zero2W (worker), Nano (relay).
func TestE2E_MultiDeviceMesh(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	url := natstest.StartJetStreamURL(ctx, t)

	nc, err := natstest.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = nc.Drain() })
	js, err := nc.JetStream()
	require.NoError(t, err)

	oraclePub, oraclePriv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	rpi5Transport, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(rpi5Transport.Close)

	zeroTransport, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(zeroTransport.Close)

	rpi5Ledger, err := effect.Open(filepath.Join(tmp, "rpi5_effect.db"))
	require.NoError(t, err)
	defer rpi5Ledger.Close()

	zeroLedger, err := effect.Open(filepath.Join(tmp, "zero_effect.db"))
	require.NoError(t, err)
	defer zeroLedger.Close()

	td := "td-e2e"
	nativeSubject := fmt.Sprintf("mesh.task.%s.native", td)
	wasmSubject := fmt.Sprintf("mesh.task.%s.wasm", td)

	task := &meshyantsv1.TaskAtom{
		SchemaVersion:    1,
		TaskId:           "task-native-1",
		IdempotencyKey:   "idem-native-1",
		Issuer:           "oracle",
		TrustDomain:      td,
		CausalParents:    []string{"genesis"},
		RequirementsJson: `{"transport":"fast-local"}`,
		Payload:          []byte(`{}`),
		ExpiresAt:        timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(task, oraclePriv))
	payload, err := proto.Marshal(task)
	require.NoError(t, err)
	_, err = js.Publish(nativeSubject, payload, nats.MsgId(task.IdempotencyKey))
	require.NoError(t, err)

	wasmTask := &meshyantsv1.TaskAtom{
		SchemaVersion:    1,
		TaskId:           "task-wasm-1",
		IdempotencyKey:   "idem-wasm-1",
		Issuer:           "oracle",
		TrustDomain:      td,
		CausalParents:    []string{"genesis"},
		RequirementsJson: `{"transport":"fast-local"}`,
		Payload:          execwasm.ValidWasm(),
		ExpiresAt:        timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(wasmTask, oraclePriv))
	wasmPayload, err := proto.Marshal(wasmTask)
	require.NoError(t, err)
	_, err = js.Publish(wasmSubject, wasmPayload, nats.MsgId(wasmTask.IdempotencyKey))
	require.NoError(t, err)

	rpi5Worker := &agent.Worker{
		Transport:    rpi5Transport,
		EffectLedger: rpi5Ledger,
		Executor:     agent.WasmExecutor{Quotas: execwasm.DefaultQuotas()},
		PublicKey:    oraclePub,
		PrivateKey:   oraclePriv,
		Issuer:       "oracle",
		TrustDomain:  td,
		Subject:      wasmSubject,
		Durable:      fmt.Sprintf("rpi5-worker-%d", time.Now().UnixNano()),
	}
	zeroWorker := &agent.Worker{
		Transport:    zeroTransport,
		EffectLedger: zeroLedger,
		Executor:     agent.NativeExecutor{},
		PublicKey:    oraclePub,
		PrivateKey:   oraclePriv,
		Issuer:       "oracle",
		TrustDomain:  td,
		Subject:      nativeSubject,
		Durable:      fmt.Sprintf("zero-worker-%d", time.Now().UnixNano()),
	}

	pheroSub := fmt.Sprintf("mesh.phero.%s", td)
	sub, err := js.PullSubscribe(pheroSub, fmt.Sprintf("phero-check-%d", time.Now().UnixNano()), nats.BindStream("MESHYANTS"), nats.DeliverAll())
	require.NoError(t, err)
	defer sub.Unsubscribe()

	rpi5Ctx, rpi5Cancel := context.WithCancel(ctx)
	defer rpi5Cancel()
	zeroCtx, zeroCancel := context.WithCancel(ctx)
	defer zeroCancel()

	errCh := make(chan error, 2)
	go func() { errCh <- rpi5Worker.Run(rpi5Ctx) }()
	go func() { errCh <- zeroWorker.Run(zeroCtx) }()

	var got []*nats.Msg
	deadline := time.Now().Add(20 * time.Second)
	for len(got) < 2 && time.Now().Before(deadline) {
		batch, err := sub.Fetch(2-len(got), nats.MaxWait(time.Second))
		if err != nil && err != nats.ErrTimeout {
			require.NoError(t, err)
		}
		got = append(got, batch...)
	}
	require.Len(t, got, 2, "expected 2 pheromones (SAFE for native + SAFE for wasm)")

	for _, msg := range got {
		var rec meshyantsv1.PheromoneRecord
		require.NoError(t, proto.Unmarshal(msg.Data, &rec))
		require.Equal(t, meshyantsv1.PheromoneKind_PHEROMONE_KIND_SAFE, rec.Kind)
	}

	rpi5Cancel()
	zeroCancel()
	for i := 0; i < 2; i++ {
		err := <-errCh
		require.ErrorIs(t, err, context.Canceled)
	}
}

// TestE2E_CapabilityRouting tests that the router selects capable nodes.
func TestE2E_CapabilityRouting(t *testing.T) {
	capStore := routing.NewCapabilityStore()
	r := routing.NewRouter(capStore)

	capStore.Set("node-1", &meshyantsv1.CapabilityAdvertisement{
		DeviceId:    "node-1",
		TrustDomain: "td",
		QueueDepth:  5,
	})
	capStore.Set("node-2", &meshyantsv1.CapabilityAdvertisement{
		DeviceId:    "node-2",
		TrustDomain: "td",
		QueueDepth:  50,
	})

	task := &meshyantsv1.TaskAtom{
		RequirementsJson: `{"transport":"fast-local"}`,
		TrustDomain:      "td",
	}
	targets, err := r.Route(task)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "node-1", targets[0])
}

// TestE2E_OracleService tests the Oracle service using a mock adapter.
func TestE2E_OracleService(t *testing.T) {
	ctx := context.Background()
	url := natstest.StartJetStreamURL(ctx, t)

	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	tr, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	defer tr.Close()

	adapter := &oracle.MockAdapter{}
	svc := oracle.NewService(adapter, tr, priv, pub, "oracle", "td-e2e")

	task, phero, err := svc.HandleGoal(ctx, "deploy my app to staging")
	require.NoError(t, err)
	require.NotNil(t, task)
	require.NotNil(t, phero)
	require.Equal(t, meshyantsv1.PheromoneKind_PHEROMONE_KIND_TODO, phero.Kind)
}
