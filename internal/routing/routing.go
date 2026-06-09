// Package routing implements capability-based routing (docs/v1/03-protocols-and-execution.md).
package routing

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/runtime/tier"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CapabilityStore holds recent CapabilityAdvertisements per node.
type CapabilityStore struct {
	mu    sync.RWMutex
	Items map[string]*meshyantsv1.CapabilityAdvertisement
}

// NewCapabilityStore creates a store.
func NewCapabilityStore() *CapabilityStore {
	return &CapabilityStore{Items: make(map[string]*meshyantsv1.CapabilityAdvertisement)}
}

// Set stores or updates an advertisement.
func (s *CapabilityStore) Set(nodeID string, cap *meshyantsv1.CapabilityAdvertisement) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Items[nodeID] = cap
}

// Get returns the advertisement for a node.
func (s *CapabilityStore) Get(nodeID string) *meshyantsv1.CapabilityAdvertisement {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Items[nodeID]
}

// Advertiser periodically broadcasts CapabilityAdvertisement records.
type Advertiser struct {
	Transport   *fabric.JetStreamTransport
	PrivateKey  ed25519.PrivateKey
	DeviceID   string
	TrustDomain string
	Interval    time.Duration
}

// DefaultAdvertiserInterval is the cadence between advertisements.
const DefaultAdvertiserInterval = 30 * time.Second

// Run starts the periodic advertisement loop. It blocks until ctx is cancelled.
func (a *Advertiser) Run(ctx context.Context) error {
	interval := a.Interval
	if interval == 0 {
		interval = DefaultAdvertiserInterval
	}
	rt := tier.Detect(ctx, tier.DefaultOptions())
	hints := &meshyantsv1.ResourceHints{
		MemoryBytes:   1024 * 1024 * 1024, // TODO: detect real memory
		CpuMillicores: 1000,
	}
	cap := &meshyantsv1.CapabilityAdvertisement{
		SchemaVersion:    1,
		DeviceId:        a.DeviceID,
		TrustDomain:     a.TrustDomain,
		RuntimeTier:     rt,
		Resources:      hints,
		Accelerators:   []string{},
		QueueDepth:     10,
		TransportProfile: string(fabric.FastWide),
		ValidUntil:    timestamppb.New(time.Now().Add(interval * 2)),
	}
	if err := signing.Sign(cap, a.PrivateKey); err != nil {
		return fmt.Errorf("sign capability: %w", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	a.publish(ctx, cap)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			cap.ValidUntil = timestamppb.New(time.Now().Add(interval * 2))
			// Re-sign with fresh timestamp.
			cap.Signature = nil
			if err := signing.Sign(cap, a.PrivateKey); err != nil {
				log.Printf("[advertiser] sign: %v", err)
				continue
			}
			a.publish(ctx, cap)
		}
	}
}

func (a *Advertiser) publish(ctx context.Context, cap *meshyantsv1.CapabilityAdvertisement) {
	payload, err := proto.Marshal(cap)
	if err != nil {
		log.Printf("[advertiser] marshal: %v", err)
		return
	}
	subject := fmt.Sprintf("mesh.cap.%s", a.TrustDomain)
	if err := a.Transport.Publish(ctx, subject, payload, fabric.PublishOpts{
		MsgID: fabric.StableMsgID("cap-" + a.DeviceID),
	}); err != nil {
		log.Printf("[advertiser] publish: %v", err)
	}
}

// Router matches incoming TaskAtoms to capable nodes with bounded-queue backpressure.
type Router struct {
	Caps *CapabilityStore
}

// RouteTarget is a candidate node for task execution.
type RouteTarget struct {
	NodeID     string
	QueueDepth uint32
	Score     float64 // higher = better
}

// NewRouter creates a router.
func NewRouter(caps *CapabilityStore) *Router {
	return &Router{Caps: caps}
}

// Route returns ranked target node IDs that satisfy the task requirements.
// Returns an error if no capable nodes are available or all are saturated.
func (r *Router) Route(task *meshyantsv1.TaskAtom) ([]string, error) {
	var req struct {
		Transport string `json:"transport"`
		Scope     string `json:"scope"`
	}
	if err := json.Unmarshal([]byte(task.RequirementsJson), &req); err != nil {
		return nil, fmt.Errorf("parse requirements: %w", err)
	}

	r.Caps.mu.RLock()
	defer r.Caps.mu.RUnlock()

	var targets []RouteTarget
	for nodeID, cap := range r.Caps.Items {
		if cap.GetTrustDomain() != task.GetTrustDomain() {
			continue
		}
		// Backpressure: skip nodes with saturated queues (>80% of max depth=50).
		if cap.QueueDepth > 40 {
			continue
		}
		targets = append(targets, RouteTarget{
			NodeID:    nodeID,
			QueueDepth: cap.QueueDepth,
			Score:     float64(50 - cap.QueueDepth), // lower queue = higher score
		})
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("routing: no capable nodes (all saturated or no capacity)")
	}

	// Sort by score descending (highest score = best).
	sortByScore(targets)

	result := make([]string, len(targets))
	for i, t := range targets {
		result[i] = t.NodeID
	}
	return result, nil
}

func sortByScore(t []RouteTarget) {
	for i := 0; i < len(t)-1; i++ {
		for j := i + 1; j < len(t); j++ {
			if t[j].Score > t[i].Score {
				t[i], t[j] = t[j], t[i]
			}
		}
	}
}
