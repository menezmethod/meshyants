// Package undertaker implements the Undertaker role: monitors the DLQ, detects failure patterns,
// and issues QuarantineNotice + DANGER pheromones (docs/v1/04-security-resilience-and-consensus.md).
package undertaker

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"strings"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/dlq"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Config defines Undertaker behavior thresholds.
type Config struct {
	// PatternWindow is the time window for pattern detection.
	PatternWindow time.Duration
	// MinQuarantineCount is the number of failures in PatternWindow before issuing quarantine.
	MinQuarantineCount int
	// DefaultQuarantineDuration is the default quarantine duration if not specified.
	DefaultQuarantineDuration time.Duration
}

// DefaultConfig returns conservative defaults.
func DefaultConfig() Config {
	return Config{
		PatternWindow:          5 * time.Minute,
		MinQuarantineCount:     3,
		DefaultQuarantineDuration: 30 * time.Minute,
	}
}

// Undertaker monitors the DLQ and issues DANGER + QuarantineNotice when bad patterns appear.
type Undertaker struct {
	DLQ           *dlq.DLQ
	Transport     *fabric.JetStreamTransport
	PrivateKey    ed25519.PrivateKey
	PublicKey     ed25519.PublicKey
	Issuer        string
	TrustDomain   string
	Config        Config
}

// NewUndertaker creates an Undertaker.
func NewUndertaker(d *dlq.DLQ, tr *fabric.JetStreamTransport, priv ed25519.PrivateKey, pub ed25519.PublicKey, issuer, td string) *Undertaker {
	return &Undertaker{
		DLQ:         d,
		Transport:   tr,
		PrivateKey:  priv,
		PublicKey:   pub,
		Issuer:      issuer,
		TrustDomain: td,
		Config:      DefaultConfig(),
	}
}

// Run starts the undertaker monitoring loop. It blocks until ctx is cancelled.
func (u *Undertaker) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := u.analyzeDLQ(ctx); err != nil {
				log.Printf("[undertaker] analyze error: %v", err)
			}
		}
	}
}

func (u *Undertaker) analyzeDLQ(ctx context.Context) error {
	entries, err := u.DLQ.ReadAll(ctx)
	if err != nil {
		return fmt.Errorf("read DLQ: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}

	// Group by issuer/subject to find repeating failure patterns.
	type issuerCount struct {
		issuer  string
		count   int
		reasons []string
	}
	patternMap := make(map[string]*issuerCount)
	window := time.Now().Add(-u.Config.PatternWindow)

	for _, e := range entries {
		if e.EnqueuedAt.Before(window) {
			continue
		}
		// Try to extract issuer from the original data.
		var task meshyantsv1.TaskAtom
		if err := proto.Unmarshal(e.OriginalData, &task); err == nil && task.Issuer != "" {
			key := task.Issuer
			if _, ok := patternMap[key]; !ok {
				patternMap[key] = &issuerCount{issuer: task.Issuer}
			}
			patternMap[key].count++
			patternMap[key].reasons = append(patternMap[key].reasons, e.Reason)
		}
	}

	// Issue quarantine if threshold exceeded.
	for _, pc := range patternMap {
		if pc.count >= u.Config.MinQuarantineCount {
			log.Printf("[undertaker] pattern detected for issuer %q: %d failures", pc.issuer, pc.count)
			reason := strings.Join(uniq(pc.reasons), "; ")
			if err := u.IssueQuarantine(ctx, pc.issuer, reason, "issuer", u.Config.DefaultQuarantineDuration); err != nil {
				log.Printf("[undertaker] issue quarantine: %v", err)
			}
		}
	}

	return nil
}

// IssueQuarantine publishes a QuarantineNotice and DANGER pheromone for the given subject.
func (u *Undertaker) IssueQuarantine(ctx context.Context, subjectDevice, reason, scope string, dur time.Duration) error {
	notice := &meshyantsv1.QuarantineNotice{
		SchemaVersion: 1,
		SubjectDevice:  subjectDevice,
		TrustDomain:    u.TrustDomain,
		Reason:         reason,
		Scope:          scope,
		Duration:       durationpb.New(dur),
		Issuer:         u.Issuer,
		IssuedAt:       timestamppb.New(time.Now().UTC()),
	}
	if err := signing.Sign(notice, u.PrivateKey); err != nil {
		return fmt.Errorf("sign quarantine notice: %w", err)
	}
	payload, err := proto.Marshal(notice)
	if err != nil {
		return fmt.Errorf("marshal notice: %w", err)
	}
	subject := fmt.Sprintf("mesh.quarantine.%s", u.TrustDomain)
	if err := u.Transport.Publish(ctx, subject, payload, fabric.PublishOpts{
		MsgID: fabric.StableMsgID("quarantine-" + subjectDevice),
	}); err != nil {
		return fmt.Errorf("publish notice: %w", err)
	}

	// Emit DANGER pheromone.
	phero := &meshyantsv1.PheromoneRecord{
		SchemaVersion: 1,
		RecordId:       fmt.Sprintf("phero-danger-%d-%s", time.Now().UnixNano(), subjectDevice),
		Kind:          meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER,
		Subject:       subjectDevice,
		Strength:      1,
		Issuer:        u.Issuer,
		TrustDomain:   u.TrustDomain,
		CausalParents: []string{},
		IssuedAt:      timestamppb.New(time.Now().UTC()),
	}
	if err := signing.Sign(phero, u.PrivateKey); err != nil {
		return fmt.Errorf("sign DANGER pheromone: %w", err)
	}
	pheroPayload, _ := proto.Marshal(phero)
	pheroSubject := fmt.Sprintf("mesh.phero.%s", u.TrustDomain)
	if err := u.Transport.Publish(ctx, pheroSubject, pheroPayload, fabric.PublishOpts{
		MsgID: fabric.StableMsgID(phero.RecordId),
	}); err != nil {
		return fmt.Errorf("publish DANGER pheromone: %w", err)
	}

	log.Printf("[undertaker] quarantine issued for %s: %s", subjectDevice, reason)
	return nil
}

// EmitDANGER emits a standalone DANGER pheromone (for use by worker on sandbox fault).
func (u *Undertaker) EmitDANGER(ctx context.Context, subject, detail string) error {
	phero := &meshyantsv1.PheromoneRecord{
		SchemaVersion: 1,
		RecordId:       fmt.Sprintf("phero-danger-%d-%s", time.Now().UnixNano(), subject),
		Kind:          meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER,
		Subject:       subject,
		Strength:      1,
		Issuer:        u.Issuer,
		TrustDomain:   u.TrustDomain,
		CausalParents: []string{subject},
		IssuedAt:      timestamppb.New(time.Now().UTC()),
	}
	_ = detail // could be included in a future extension record
	if err := signing.Sign(phero, u.PrivateKey); err != nil {
		return err
	}
	payload, _ := proto.Marshal(phero)
	pheroSubject := fmt.Sprintf("mesh.phero.%s", u.TrustDomain)
	return u.Transport.Publish(ctx, pheroSubject, payload, fabric.PublishOpts{
		MsgID: fabric.StableMsgID(phero.RecordId),
	})
}

func uniq(ss []string) []string {
	seen := make(map[string]bool)
	out := []string{}
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
