// Package validate enforces V1 contract rules from docs/v1/03-protocols-and-execution.md.
package validate

import (
	"errors"
	"fmt"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	MinSchemaVersion = 1
	// DTNMaxTaskAtomWireBytes is the TaskAtom wire size budget excluding the signature field (delay-tolerant lane).
	DTNMaxTaskAtomWireBytes = 2048
)

var (
	ErrValidation = errors.New("validate: contract validation failed")
)

func nonEmpty(s, name string) error {
	if s == "" {
		return fmt.Errorf("%w: missing %s", ErrValidation, name)
	}
	return nil
}

// JoinGrant checks required fields (signature verified separately).
func JoinGrant(g *meshyantsv1.JoinGrant) error {
	if g == nil {
		return fmt.Errorf("%w: nil JoinGrant", ErrValidation)
	}
	if g.SchemaVersion < MinSchemaVersion {
		return fmt.Errorf("%w: schema_version", ErrValidation)
	}
	if err := nonEmpty(g.GrantId, "grant_id"); err != nil {
		return err
	}
	if err := nonEmpty(g.TrustDomain, "trust_domain"); err != nil {
		return err
	}
	if err := nonEmpty(g.Issuer, "issuer"); err != nil {
		return err
	}
	if g.ExpiresAt == nil {
		return fmt.Errorf("%w: expires_at required", ErrValidation)
	}
	if g.ExpiresAt.AsTime().Before(time.Now()) {
		return fmt.Errorf("%w: expires_at in the past", ErrValidation)
	}
	if len(g.Signature) == 0 {
		return fmt.Errorf("%w: signature required", ErrValidation)
	}
	return nil
}

// ProvisioningManifest checks required fields.
func ProvisioningManifest(m *meshyantsv1.ProvisioningManifest) error {
	if m == nil {
		return fmt.Errorf("%w: nil ProvisioningManifest", ErrValidation)
	}
	if m.SchemaVersion < MinSchemaVersion {
		return fmt.Errorf("%w: schema_version", ErrValidation)
	}
	if err := nonEmpty(m.DeviceId, "device_id"); err != nil {
		return err
	}
	if err := nonEmpty(m.TrustDomain, "trust_domain"); err != nil {
		return err
	}
	if m.RuntimeTier == meshyantsv1.RuntimeTier_RUNTIME_TIER_UNSPECIFIED {
		return fmt.Errorf("%w: runtime_tier", ErrValidation)
	}
	if err := nonEmpty(m.Issuer, "issuer"); err != nil {
		return err
	}
	if m.ExpiresAt == nil {
		return fmt.Errorf("%w: expires_at required", ErrValidation)
	}
	if len(m.Signature) == 0 {
		return fmt.Errorf("%w: signature required", ErrValidation)
	}
	return nil
}

// TaskAtom checks required fields and optional DTN wire budget (excluding signature).
func TaskAtom(t *meshyantsv1.TaskAtom, dtn bool) error {
	if t == nil {
		return fmt.Errorf("%w: nil TaskAtom", ErrValidation)
	}
	if t.SchemaVersion < MinSchemaVersion {
		return fmt.Errorf("%w: schema_version", ErrValidation)
	}
	if err := nonEmpty(t.TaskId, "task_id"); err != nil {
		return err
	}
	if err := nonEmpty(t.IdempotencyKey, "idempotency_key"); err != nil {
		return err
	}
	if err := nonEmpty(t.Issuer, "issuer"); err != nil {
		return err
	}
	if err := nonEmpty(t.TrustDomain, "trust_domain"); err != nil {
		return err
	}
	if len(t.CausalParents) == 0 {
		return fmt.Errorf("%w: causal_parents required", ErrValidation)
	}
	if t.ExpiresAt == nil {
		return fmt.Errorf("%w: expires_at required", ErrValidation)
	}
	if len(t.Signature) == 0 {
		return fmt.Errorf("%w: signature required", ErrValidation)
	}
	hasBody := len(t.Payload) > 0 || t.ContentRef != ""
	if !hasBody {
		return fmt.Errorf("%w: payload or content_ref required", ErrValidation)
	}
	if dtn {
		clone := proto.Clone(t).(*meshyantsv1.TaskAtom)
		clone.Signature = nil
		wire, err := proto.Marshal(clone)
		if err != nil {
			return err
		}
		if len(wire) > DTNMaxTaskAtomWireBytes {
			return fmt.Errorf("%w: TaskAtom exceeds DTN budget (%d > %d bytes)", ErrValidation, len(wire), DTNMaxTaskAtomWireBytes)
		}
	}
	return nil
}

// PheromoneRecord checks required fields.
func PheromoneRecord(p *meshyantsv1.PheromoneRecord) error {
	if p == nil {
		return fmt.Errorf("%w: nil PheromoneRecord", ErrValidation)
	}
	if p.SchemaVersion < MinSchemaVersion {
		return fmt.Errorf("%w: schema_version", ErrValidation)
	}
	if err := nonEmpty(p.RecordId, "record_id"); err != nil {
		return err
	}
	if p.Kind == meshyantsv1.PheromoneKind_PHEROMONE_KIND_UNSPECIFIED {
		return fmt.Errorf("%w: kind", ErrValidation)
	}
	if err := nonEmpty(p.Subject, "subject"); err != nil {
		return err
	}
	if err := nonEmpty(p.Issuer, "issuer"); err != nil {
		return err
	}
	if err := nonEmpty(p.TrustDomain, "trust_domain"); err != nil {
		return err
	}
	if p.IssuedAt == nil {
		return fmt.Errorf("%w: issued_at required", ErrValidation)
	}
	if len(p.Signature) == 0 {
		return fmt.Errorf("%w: signature required", ErrValidation)
	}
	return nil
}

// UpdateManifest checks required fields for rollout preflight.
func UpdateManifest(u *meshyantsv1.UpdateManifest) error {
	if u == nil {
		return fmt.Errorf("%w: nil UpdateManifest", ErrValidation)
	}
	if u.SchemaVersion < MinSchemaVersion {
		return fmt.Errorf("%w: schema_version", ErrValidation)
	}
	if err := nonEmpty(u.ReleaseId, "release_id"); err != nil {
		return err
	}
	if err := nonEmpty(u.TrustDomain, "trust_domain"); err != nil {
		return err
	}
	if len(u.Targets) == 0 {
		return fmt.Errorf("%w: targets required", ErrValidation)
	}
	if err := nonEmpty(u.CompatWindow, "compat_window"); err != nil {
		return err
	}
	if len(u.Hashes) == 0 {
		return fmt.Errorf("%w: hashes required", ErrValidation)
	}
	if err := nonEmpty(u.Issuer, "issuer"); err != nil {
		return err
	}
	if u.ValidUntil == nil {
		return fmt.Errorf("%w: valid_until required", ErrValidation)
	}
	if u.ValidUntil.AsTime().Before(time.Now()) {
		return fmt.Errorf("%w: valid_until in the past", ErrValidation)
	}
	if len(u.Signature) == 0 {
		return fmt.Errorf("%w: signature required", ErrValidation)
	}
	return nil
}

// CapabilityAdvertisement validates capability ads.
func CapabilityAdvertisement(c *meshyantsv1.CapabilityAdvertisement) error {
	if c == nil {
		return fmt.Errorf("%w: nil CapabilityAdvertisement", ErrValidation)
	}
	if c.SchemaVersion < MinSchemaVersion {
		return fmt.Errorf("%w: schema_version", ErrValidation)
	}
	if err := nonEmpty(c.DeviceId, "device_id"); err != nil {
		return err
	}
	if err := nonEmpty(c.TrustDomain, "trust_domain"); err != nil {
		return err
	}
	if c.RuntimeTier == meshyantsv1.RuntimeTier_RUNTIME_TIER_UNSPECIFIED {
		return fmt.Errorf("%w: runtime_tier", ErrValidation)
	}
	if err := nonEmpty(c.TransportProfile, "transport_profile"); err != nil {
		return err
	}
	if c.ValidUntil == nil {
		return fmt.Errorf("%w: valid_until required", ErrValidation)
	}
	if len(c.Signature) == 0 {
		return fmt.Errorf("%w: signature required", ErrValidation)
	}
	return nil
}

// TimestampNow is a test helper hook; production code should use real clock injection.
func TimestampNow() *timestamppb.Timestamp {
	return timestamppb.New(time.Now().UTC())
}
