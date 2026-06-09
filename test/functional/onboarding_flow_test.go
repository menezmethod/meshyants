//go:build integration

package functional_test

import (
	"context"
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/meshyants/meshyants/v1/internal/onboarding"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Admission through signed manifests and one publish to blackboard (operator-visible path uses fabric).
func TestFunctional_AdmitAndPublishCapability(t *testing.T) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"-js"},
		WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(60 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Terminate(ctx) })
	host, err := c.Host(ctx)
	require.NoError(t, err)
	port, err := c.MappedPort(ctx, "4222")
	require.NoError(t, err)
	url := "nats://" + host + ":" + port.Port()

	pub, priv, err := signing.GenerateKeyPair()
	require.NoError(t, err)
	g := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		GrantId:       "fg-1",
		TrustDomain:   "td-func",
		Issuer:        "queen",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(g, priv))
	s := onboarding.NewSession()
	require.NoError(t, s.ApplyJoinGrant(g))
	require.NoError(t, s.ConfirmLocal())
	m := &meshyantsv1.ProvisioningManifest{
		SchemaVersion: 1,
		DeviceId:      "dev-func",
		TrustDomain:   "td-func",
		RuntimeTier:   meshyantsv1.RuntimeTier_RUNTIME_TIER_RELAY,
		Issuer:        "queen",
		BuildPolicy:   "dev",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(m, priv))
	require.NoError(t, s.ApplyProvisioningManifest(m, nil))

	cap := &meshyantsv1.CapabilityAdvertisement{
		SchemaVersion:    1,
		DeviceId:         m.DeviceId,
		TrustDomain:      m.TrustDomain,
		RuntimeTier:      m.RuntimeTier,
		TransportProfile: "fast-wide",
		ValidUntil:       timestamppb.New(time.Now().Add(time.Hour)),
	}
	require.NoError(t, signing.Sign(cap, priv))
	require.NoError(t, signing.Verify(cap, pub))

	tr, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(tr.Close)
	body, err := proto.Marshal(cap)
	require.NoError(t, err)
	err = tr.Publish(ctx, "mesh.capability."+m.DeviceId, body, fabric.PublishOpts{MsgID: fabric.StableMsgID("cap-" + m.DeviceId)})
	require.NoError(t, err)
}
