// Package sim provides a multi-node mesh simulator for QA testing.
// It simulates LicheeRV Nano, RPi Zero 2W, and RPi 5 as distinct runtime tiers
// communicating over JetStream (docs/v1/01-platform-architecture.md).
package sim

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"sync"
	"testing"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/agent"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/ledger/effect"
	"github.com/meshyants/meshyants/v1/internal/routing"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// DeviceType models a specific hardware target.
type DeviceType string

const (
	// Nano: LicheeRV Nano — ~256MB RAM, single-core, no Wasm (RELAY or EXECUTOR_BASIC only).
	Nano DeviceType = "nano"
	// Zero2W: Raspberry Pi Zero 2W — ~512MB RAM, 32-bit ARM, limited Wasm.
	Zero2W DeviceType = "zero2w"
	// RPi5: Raspberry Pi 5 — 8GB RAM, 64-bit ARM, full Wasm support.
	RPi5 DeviceType = "rpi5"
)

// SimNode is a simulated mesh node.
type SimNode struct {
	ID          string
	Type        DeviceType
	TrustDomain string
	PubKey      ed25519.PublicKey
	PrivKey     ed25519.PrivateKey
	Issuer      string
	Transport   *fabric.JetStreamTransport
	EffectLedger *effect.Ledger
	Worker      *agent.Worker
	CapStore    *routing.CapabilityStore
	Ready       chan struct{}
	ErrCh       chan error
}

// Simulator manages a collection of SimNodes on a shared JetStream connection.
type Simulator struct {
	T      testing.TB
	NC     *nats.Conn
	JS     nats.JetStreamContext
	Nodes  map[string]*SimNode
	mu     sync.Mutex
	tmpDir string
}

// NewSimulator creates a simulator with NATS connection and a test name.
func NewSimulator(t testing.TB, nc *nats.Conn) *Simulator {
	js, err := nc.JetStream()
	require.NoError(t, err)
	return &Simulator{
		T:     t,
		NC:    nc,
		JS:    js,
		Nodes: make(map[string]*SimNode),
	}
}

// AddNode creates and registers a new simulated node.
func (s *Simulator) AddNode(id string, dt DeviceType, td string) *SimNode {
	s.mu.Lock()
	defer s.mu.Unlock()

	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(s.T, err)

	transport, err := fabric.ConnectJetStream(context.Background(), s.NC.Opts.Url, fabric.DefaultJetStreamConfig())
	require.NoError(s.T, err)

	effLedger, err := effect.Open(fmt.Sprintf("/tmp/meshyants_sim_%s.db", id))
	require.NoError(s.T, err)

	node := &SimNode{
		ID:           id,
		Type:         dt,
		TrustDomain:  td,
		PubKey:      pub,
		PrivKey:     priv,
		Issuer:      id,
		Transport:   transport,
		EffectLedger: effLedger,
		CapStore:    routing.NewCapabilityStore(),
		Ready:        make(chan struct{}),
		ErrCh:        make(chan error, 1),
	}
	s.Nodes[id] = node
	return node
}

// StartWorker starts the agent worker loop for a node.
func (s *Simulator) StartWorker(ctx context.Context, node *SimNode, subject string) {
	go func() {
		w := &agent.Worker{
			Transport:    node.Transport,
			EffectLedger: node.EffectLedger,
			Executor:     node.Executor(),
			PublicKey:    node.PubKey,
			PrivateKey:   node.PrivKey,
			Issuer:       node.Issuer,
			TrustDomain:  node.TrustDomain,
			Subject:      subject,
			Durable:      "sim-" + node.ID,
		}
		node.Worker = w
		close(node.Ready)
		node.ErrCh <- w.Run(ctx)
	}()
}

// Executor returns the appropriate executor for the device type.
func (n *SimNode) Executor() agent.Executor {
	switch n.Type {
	case Nano, Zero2W:
		return agent.NativeExecutor{}
	case RPi5:
		return agent.WasmExecutor{}
	default:
		return agent.NativeExecutor{}
	}
}

// RuntimeTier returns the MeshyAnts runtime tier for the device type.
func (n *SimNode) RuntimeTier() meshyantsv1.RuntimeTier {
	switch n.Type {
	case Nano:
		return meshyantsv1.RuntimeTier_RUNTIME_TIER_RELAY
	case Zero2W:
		return meshyantsv1.RuntimeTier_RUNTIME_TIER_EXECUTOR_BASIC
	case RPi5:
		return meshyantsv1.RuntimeTier_RUNTIME_TIER_EXECUTOR_WASM
	default:
		return meshyantsv1.RuntimeTier_RUNTIME_TIER_RELAY
	}
}

// PublishTask publishes a signed task from this node.
func (s *Simulator) PublishTask(node *SimNode, subject string, task *meshyantsv1.TaskAtom) {
	require.NoError(s.T, signing.Sign(task, node.PrivKey))
	payload, err := proto.Marshal(task)
	require.NoError(s.T, err)
	err = node.Transport.Publish(context.Background(), subject, payload, fabric.PublishOpts{
		MsgID: fabric.StableMsgID(task.IdempotencyKey),
	})
	require.NoError(s.T, err)
}

// Cleanup stops all nodes and closes resources.
func (s *Simulator) Cleanup() {
	for _, n := range s.Nodes {
		if n.Worker != nil {
			// Worker doesn't have a stop channel, but closing transport stops it.
		}
		n.Transport.Close()
		n.EffectLedger.Close()
	}
}
