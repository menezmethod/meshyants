package signing_test

import (
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSignVerify_RoundTrip(t *testing.T) {
	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)
	m := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		GrantId:       "g1",
		TrustDomain:   "td",
		Issuer:        "queen",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(m, priv))
	require.NoError(t, signing.Verify(m, pub))
	m.Signature = nil
	require.Error(t, signing.Verify(m, pub))
}
