// Package oracle provides LLM-backed Oracle Interface Agents.
package oracle

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/oracle/policy"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service orchestrates the Oracle Interface Agent lifecycle.
type Service struct {
	Adapter     Adapter
	Transport   *fabric.JetStreamTransport
	PrivateKey  ed25519.PrivateKey
	PublicKey   ed25519.PublicKey
	Issuer      string
	TrustDomain string
	// Verbose logs the fixed canonicalization system prompt, raw user goal, and parsed TaskIntentHeader.
	Verbose bool
}

// NewService creates an Oracle Service.
func NewService(adapter Adapter, tr *fabric.JetStreamTransport, priv ed25519.PrivateKey, pub ed25519.PublicKey, issuer, td string) *Service {
	return &Service{
		Adapter:     adapter,
		Transport:   tr,
		PrivateKey:  priv,
		PublicKey:   pub,
		Issuer:      issuer,
		TrustDomain: td,
	}
}

// HandleGoal translates a human goal into a signed TaskAtom and TODO pheromone, then publishes both.
// It implements the canonicalization → signing → publish path from docs/v1/03.
func (s *Service) HandleGoal(ctx context.Context, rawInput string) (*meshyantsv1.TaskAtom, *meshyantsv1.PheromoneRecord, error) {
	// 1. Canonicalize: LLM extracts structured intent header.
	header, err := s.Adapter.Canonicalize(ctx, rawInput)
	if err != nil {
		return nil, nil, fmt.Errorf("canonicalize: %w", err)
	}
	if s.Verbose {
		hdrJSON, _ := json.MarshalIndent(header, "", "  ")
		log.Printf("[oracle] canonicalization system prompt (fixed template, not model-generated):\n%s", TaskIntentCanonicalizeSystemPrompt)
		log.Printf("[oracle] user message (your goal):\n%s", rawInput)
		log.Printf("[oracle] TaskIntentHeader (model structured output):\n%s", string(hdrJSON))
		log.Printf("[oracle] API constraint: adapters send response_format=json_schema (TaskIntentHeader, strict); schema source: internal/oracle/intent_schema.json")
	}

	// 2. Check DTN size budget.
	canonical, err := policy.CheckDTNSize(header, false) // false = reject on overflow
	if err != nil {
		// Return the header with overflow signal rather than dropping the goal.
		log.Printf("[oracle] goal exceeds DTN budget: %v", err)
		return nil, nil, fmt.Errorf("goal exceeds DTN budget: %w", err)
	}

	// 3. Derive stable idempotency key from canonical goal + interaction ID.
	clientID := uuid.New().String()
	idemKey := policy.DeriveIdempotencyKey(header.CanonicalGoal, clientID)

	// 4. Build TaskAtom.
	deadline := time.Now().Add(24 * time.Hour)
	if header.DeadlineRFC3339 != "" {
		if d, err := time.Parse(time.RFC3339, header.DeadlineRFC3339); err == nil {
			deadline = d
		}
	}

	requirements := map[string]any{
		"transport": string(header.TransportClass),
		"scope":     header.Scope,
	}
	reqJSON, _ := json.Marshal(requirements)

	task := &meshyantsv1.TaskAtom{
		SchemaVersion:    1,
		TaskId:          uuid.New().String(),
		IdempotencyKey:  idemKey,
		Issuer:          s.Issuer,
		TrustDomain:     s.TrustDomain,
		CausalParents:   []string{"genesis"},
		RequirementsJson: string(reqJSON),
		Payload:         canonical,
		ExpiresAt:       timestamppb.New(deadline),
	}

	// 5. Sign and publish TaskAtom.
	if err := signing.Sign(task, s.PrivateKey); err != nil {
		return nil, nil, fmt.Errorf("sign task: %w", err)
	}
	taskSubject := fmt.Sprintf("mesh.task.%s", s.TrustDomain)
	payload, err := proto.Marshal(task)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal task: %w", err)
	}
	if err := s.Transport.Publish(ctx, taskSubject, payload, fabric.PublishOpts{
		MsgID: fabric.StableMsgID(idemKey),
	}); err != nil {
		return nil, nil, fmt.Errorf("publish task: %w", err)
	}

	// 6. Emit TODO pheromone alongside the TaskAtom (docs/v1/03: Oracle must emit TODO pheromone).
	phero := &meshyantsv1.PheromoneRecord{
		SchemaVersion: 1,
		RecordId:       fmt.Sprintf("phero-todo-%d-%s", time.Now().UnixNano(), task.TaskId),
		Kind:          meshyantsv1.PheromoneKind_PHEROMONE_KIND_TODO,
		Subject:        task.TaskId,
		Strength:      1,
		Issuer:        s.Issuer,
		TrustDomain:   s.TrustDomain,
		CausalParents: []string{task.TaskId},
		IssuedAt:      timestamppb.New(time.Now().UTC()),
	}
	if err := signing.Sign(phero, s.PrivateKey); err != nil {
		return nil, nil, fmt.Errorf("sign pheromone: %w", err)
	}
	pheroPayload, err := proto.Marshal(phero)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal pheromone: %w", err)
	}
	pheroSubject := fmt.Sprintf("mesh.phero.%s", s.TrustDomain)
	if err := s.Transport.Publish(ctx, pheroSubject, pheroPayload, fabric.PublishOpts{
		MsgID: fabric.StableMsgID(phero.RecordId),
	}); err != nil {
		return nil, nil, fmt.Errorf("publish pheromone: %w", err)
	}

	return task, phero, nil
}

// MockAdapter is a deterministic adapter for testing and dev.
type MockAdapter struct {
	Header *policy.TaskIntentHeader
	Err    error
}

func (m *MockAdapter) Canonicalize(ctx context.Context, goal string) (*policy.TaskIntentHeader, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if m.Header != nil {
		return m.Header, nil
	}
	return &policy.TaskIntentHeader{
		Scope:             goal,
		ProhibitedActions: []string{},
		DeadlineRFC3339:   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		TransportClass:    "fast-local",
		CanonicalGoal:     goal,
	}, nil
}

func (m *MockAdapter) Summarize(ctx context.Context, records []*meshyantsv1.PheromoneRecord) (string, error) {
	return fmt.Sprintf("swarm has %d records", len(records)), nil
}
