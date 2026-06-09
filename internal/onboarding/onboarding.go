// Package onboarding models the join state machine (docs/v1/02-installation-onboarding-and-governance.md).
package onboarding

import (
	"errors"
	"fmt"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/validate"
)

// AdmissionState is operator-visible admission status.
type AdmissionState string

const (
	StatePending    AdmissionState = "pending"
	StateWarmup     AdmissionState = "warmup"
	StateNormal     AdmissionState = "normal"
	StateQuarantine AdmissionState = "quarantined"
	StateRevoked    AdmissionState = "revoked"
)

var (
	ErrOutOfOrder     = errors.New("onboarding: step out of order")
	ErrNotConfirmed   = errors.New("onboarding: local confirmation required")
	ErrGrantConsumed  = errors.New("onboarding: join grant already applied")
	ErrIdentityClash  = errors.New("onboarding: duplicate device identity in trust domain")
	ErrTrustMismatch  = errors.New("onboarding: trust_domain mismatch")
)

// Session tracks onboarding for one device key.
type Session struct {
	State           AdmissionState
	TrustDomain     string
	DeviceID        string
	grantSeen       bool
	localConfirmed  bool
	manifestApplied bool
	// InstanceNonce distinguishes VM clones when device_id collides (audit I7 hook).
	InstanceNonce string
}

// NewSession starts in pending state.
func NewSession() *Session {
	return &Session{State: StatePending}
}

// ApplyJoinGrant validates and records the one-time grant.
func (s *Session) ApplyJoinGrant(g *meshyantsv1.JoinGrant) error {
	if s == nil {
		return fmt.Errorf("onboarding: nil session")
	}
	if s.grantSeen {
		return ErrGrantConsumed
	}
	if err := validate.JoinGrant(g); err != nil {
		return err
	}
	s.grantSeen = true
	s.TrustDomain = g.TrustDomain
	return nil
}

// ConfirmLocal records production human confirmation (button, console, etc.).
func (s *Session) ConfirmLocal() error {
	if s == nil {
		return ErrOutOfOrder
	}
	if !s.grantSeen {
		return ErrOutOfOrder
	}
	s.localConfirmed = true
	return nil
}

// ApplyProvisioningManifest validates manifest and moves to warmup.
func (s *Session) ApplyProvisioningManifest(m *meshyantsv1.ProvisioningManifest, existingDeviceNonces map[string]string) error {
	if s == nil {
		return ErrOutOfOrder
	}
	if !s.localConfirmed {
		return ErrNotConfirmed
	}
	if err := validate.ProvisioningManifest(m); err != nil {
		return err
	}
	if s.TrustDomain != "" && m.TrustDomain != s.TrustDomain {
		return ErrTrustMismatch
	}
	if existingDeviceNonces != nil {
		if prev, ok := existingDeviceNonces[m.DeviceId]; ok && prev != s.InstanceNonce {
			return ErrIdentityClash
		}
	}
	s.DeviceID = m.DeviceId
	s.manifestApplied = true
	s.State = StateWarmup
	return nil
}

// PromoteToNormal moves node out of warmup after policy timers pass (caller-driven).
func (s *Session) PromoteToNormal() error {
	if s == nil || !s.manifestApplied {
		return ErrOutOfOrder
	}
	s.State = StateNormal
	return nil
}

// Quarantine marks containment.
func (s *Session) Quarantine() {
	if s == nil {
		return
	}
	s.State = StateQuarantine
}
