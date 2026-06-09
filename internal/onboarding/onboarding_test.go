package onboarding_test

import (
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/onboarding"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestOnboarding_HappyPath(t *testing.T) {
	t.Parallel()
	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	g := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		GrantId:       "grant-1",
		TrustDomain:   "td-dev",
		Issuer:        "queen",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(g, priv))
	require.NoError(t, signing.Verify(g, pub))

	s := onboarding.NewSession()
	require.NoError(t, s.ApplyJoinGrant(g))
	require.NoError(t, s.ConfirmLocal())

	m := &meshyantsv1.ProvisioningManifest{
		SchemaVersion: 1,
		DeviceId:      "device-1",
		TrustDomain:   "td-dev",
		RuntimeTier:   meshyantsv1.RuntimeTier_RUNTIME_TIER_RELAY,
		Issuer:        "queen",
		BuildPolicy:   "dev",
		ExpiresAt:     timestamppb.New(time.Now().Add(24 * time.Hour)),
	}
	require.NoError(t, signing.Sign(m, priv))
	require.NoError(t, s.ApplyProvisioningManifest(m, nil))
	require.NoError(t, s.PromoteToNormal())
	require.Equal(t, onboarding.StateNormal, s.State)
}

func TestOnboarding_I7_DuplicateIdentity(t *testing.T) {
	t.Parallel()
	_, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)

	g := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		GrantId:       "g2",
		TrustDomain:   "td",
		Issuer:        "q",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(g, priv))

	s := onboarding.NewSession()
	require.NoError(t, s.ApplyJoinGrant(g))
	require.NoError(t, s.ConfirmLocal())
	s.InstanceNonce = "nonce-a"

	m := &meshyantsv1.ProvisioningManifest{
		SchemaVersion: 1,
		DeviceId:      "dup",
		TrustDomain:   "td",
		RuntimeTier:   meshyantsv1.RuntimeTier_RUNTIME_TIER_RELAY,
		Issuer:        "q",
		BuildPolicy:   "dev",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(m, priv))

	existing := map[string]string{"dup": "nonce-b"}
	require.ErrorIs(t, s.ApplyProvisioningManifest(m, existing), onboarding.ErrIdentityClash)
}

func TestOnboarding_TrustMismatch(t *testing.T) {
	t.Parallel()
	_, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)
	g := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		GrantId:       "g3",
		TrustDomain:   "td-a",
		Issuer:        "q",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(g, priv))
	s := onboarding.NewSession()
	require.NoError(t, s.ApplyJoinGrant(g))
	require.NoError(t, s.ConfirmLocal())
	m := &meshyantsv1.ProvisioningManifest{
		SchemaVersion: 1,
		DeviceId:      "d",
		TrustDomain:   "td-b",
		RuntimeTier:   meshyantsv1.RuntimeTier_RUNTIME_TIER_RELAY,
		Issuer:        "q",
		BuildPolicy:   "dev",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(m, priv))
	require.ErrorIs(t, s.ApplyProvisioningManifest(m, nil), onboarding.ErrTrustMismatch)
}
