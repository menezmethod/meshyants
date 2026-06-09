//go:build integration

package fabric_test

import (
	"context"
	"testing"
	"time"

	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestJetStreamPublishStableMsgID audit: I2 — async-style publish with stable MsgId dedupe.
func TestJetStreamPublishStableMsgID(t *testing.T) {
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

	tr, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(tr.Close)

	idem := "idem-consistent-1"
	msgID := fabric.StableMsgID(idem)
	subj := "mesh.tasks.test"
	payload := []byte(`{"k":1}`)

	err = tr.Publish(ctx, subj, payload, fabric.PublishOpts{MsgID: msgID})
	require.NoError(t, err)
	err = tr.Publish(ctx, subj, payload, fabric.PublishOpts{MsgID: msgID})
	require.NoError(t, err)
}

// TestJetStreamReconnectUsesSameDedupe audit: I6 baseline — new connection, same MsgId should dedupe within window.
func TestJetStreamReconnectUsesSameDedupe(t *testing.T) {
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

	msgID := fabric.StableMsgID("reconnect-key")
	subj := "mesh.tasks.reconnect"
	data := []byte("x")

	tr1, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	require.NoError(t, tr1.Publish(ctx, subj, data, fabric.PublishOpts{MsgID: msgID}))
	tr1.Close()

	time.Sleep(100 * time.Millisecond)

	tr2, err := fabric.ConnectJetStream(ctx, url, fabric.DefaultJetStreamConfig())
	require.NoError(t, err)
	t.Cleanup(tr2.Close)
	require.NoError(t, tr2.Publish(ctx, subj, data, fabric.PublishOpts{MsgID: msgID}))
}
