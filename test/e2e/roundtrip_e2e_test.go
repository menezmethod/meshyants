//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/agent"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/meshyants/meshyants/v1/internal/oracle"
	"github.com/meshyants/meshyants/v1/internal/oracleview"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/meshyants/meshyants/v1/test/natstest"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// audit: I1 — signed TaskAtom and pheromones through blackboard without side channel.
func TestE2E_TaskAndPheromoneRoundTrip(t *testing.T) {
	ctx := context.Background()
	url := natstest.StartJetStreamURL(ctx, t)

	tr, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(tr.Close)

	nc, err := natstest.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = nc.Drain() })
	js, err := nc.JetStream()
	require.NoError(t, err)

	durable := fmt.Sprintf("e2e%d", time.Now().UnixNano())
	sub, err := js.PullSubscribe("mesh.e2e.task", durable, nats.BindStream("MESHYANTS"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })

	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	task := &meshyantsv1.TaskAtom{
		SchemaVersion:    1,
		TaskId:           "task-e2e-1",
		IdempotencyKey:   "idem-e2e",
		Issuer:           "oracle",
		TrustDomain:      "td-e2e",
		CausalParents:    []string{"genesis"},
		RequirementsJson: `{"transport":"fast-wide"}`,
		Payload:          []byte(`{"op":"echo"}`),
		ExpiresAt:        timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(task, priv))
	todo := &meshyantsv1.PheromoneRecord{
		SchemaVersion: 1,
		RecordId:      "todo-1",
		Kind:          meshyantsv1.PheromoneKind_PHEROMONE_KIND_TODO,
		Subject:       task.TaskId,
		Strength:      1,
		Issuer:        "oracle",
		TrustDomain:   "td-e2e",
		CausalParents: []string{task.TaskId},
		IssuedAt:      timestamppb.New(time.Now().UTC()),
	}
	require.NoError(t, signing.Sign(todo, priv))

	tb, err := proto.Marshal(task)
	require.NoError(t, err)
	require.NoError(t, tr.Publish(ctx, "mesh.e2e.task", tb, fabric.PublishOpts{MsgID: fabric.StableMsgID(task.IdempotencyKey)}))

	msgs, err := sub.Fetch(1, nats.MaxWait(10*time.Second))
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	var got meshyantsv1.TaskAtom
	require.NoError(t, proto.Unmarshal(msgs[0].Data, &got))
	require.NoError(t, signing.Verify(&got, pub))
	require.Equal(t, task.TaskId, got.TaskId)
	require.NoError(t, msgs[0].Ack())

	safe := &meshyantsv1.PheromoneRecord{
		SchemaVersion: 1,
		RecordId:      "safe-1",
		Kind:          meshyantsv1.PheromoneKind_PHEROMONE_KIND_SAFE,
		Subject:       task.TaskId,
		Strength:      1,
		Issuer:        "worker",
		TrustDomain:   "td-e2e",
		CausalParents: []string{task.TaskId},
		IssuedAt:      timestamppb.New(time.Now().UTC()),
	}
	require.NoError(t, signing.Sign(safe, priv))
	views, err := oracleview.FormatPheromones([]*meshyantsv1.PheromoneRecord{safe, todo})
	require.NoError(t, err)
	require.Len(t, views, 2)
}

// TestE2E_OracleWorkerClosedLoop: mock oracle publishes TaskAtom + TODO; worker consumes task and emits SAFE.
func TestE2E_OracleWorkerClosedLoop(t *testing.T) {
	ctx := context.Background()
	url := natstest.StartJetStreamURL(ctx, t)

	td := fmt.Sprintf("tdloop%d", time.Now().UnixNano())
	taskSubject := fmt.Sprintf("mesh.task.%s", td)
	pheroSubject := fmt.Sprintf("mesh.phero.%s", td)

	trOracle, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(trOracle.Close)

	trWorker, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(trWorker.Close)

	nc, err := natstest.Connect(ctx, url)
	require.NoError(t, err)
	t.Cleanup(func() { _ = nc.Drain() })
	js, err := nc.JetStream()
	require.NoError(t, err)

	oraclePub, oraclePriv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	ledger, err := effect.Open(filepath.Join(t.TempDir(), "effect.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = ledger.Close() })

	w := &agent.Worker{
		Transport:    trWorker,
		EffectLedger: ledger,
		Executor:     agent.NativeExecutor{},
		PublicKey:    oraclePub,
		PrivateKey:   oraclePriv,
		Issuer:       "oracle",
		TrustDomain:  td,
		Subject:      taskSubject,
		Durable:      fmt.Sprintf("e2e-loop-%d", time.Now().UnixNano()),
	}

	witnessDurable := fmt.Sprintf("e2e-witness-%d", time.Now().UnixNano())
	pheroSub, err := js.PullSubscribe(pheroSubject, witnessDurable, nats.BindStream("MESHYANTS"), nats.DeliverAll())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pheroSub.Unsubscribe() })

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	errCh := make(chan error, 1)
	go func() { errCh <- w.Run(runCtx) }()

	time.Sleep(300 * time.Millisecond)

	svc := oracle.NewService(&oracle.MockAdapter{}, trOracle, oraclePriv, oraclePub, "oracle", td)
	task, todoPhero, err := svc.HandleGoal(ctx, "deploy my app to staging")
	require.NoError(t, err)
	require.NotNil(t, task)
	require.NotNil(t, todoPhero)

	var got []*nats.Msg
	deadline := time.Now().Add(20 * time.Second)
	for len(got) < 2 && time.Now().Before(deadline) {
		batch, err := pheroSub.Fetch(2-len(got), nats.MaxWait(time.Second))
		if err != nil && err != nats.ErrTimeout {
			require.NoError(t, err)
		}
		got = append(got, batch...)
	}
	require.Len(t, got, 2, "want TODO from oracle and SAFE from worker")

	var kinds []meshyantsv1.PheromoneKind
	for _, m := range got {
		var rec meshyantsv1.PheromoneRecord
		require.NoError(t, proto.Unmarshal(m.Data, &rec))
		require.NoError(t, signing.Verify(&rec, oraclePub))
		require.Equal(t, task.TaskId, rec.Subject)
		kinds = append(kinds, rec.Kind)
	}
	require.Contains(t, kinds, meshyantsv1.PheromoneKind_PHEROMONE_KIND_TODO)
	require.Contains(t, kinds, meshyantsv1.PheromoneKind_PHEROMONE_KIND_SAFE)

	cancelRun()
	err = <-errCh
	require.ErrorIs(t, err, context.Canceled)
}
