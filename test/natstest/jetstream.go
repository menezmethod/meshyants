//go:build e2e || integration

// Package natstest starts NATS+JetStream via testcontainers for e2e and integration tests.
package natstest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartJetStreamURL starts a NATS container with JetStream enabled and returns nats://host:port.
// The container is terminated via t.Cleanup.
func StartJetStreamURL(ctx context.Context, t *testing.T) string {
	t.Helper()
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
	return "nats://" + host + ":" + port.Port()
}

// Connect dials NATS with retries. Mapped ports can lag the container "ready" log slightly,
// which matters when many packages start brokers in parallel under `go test ./...`.
func Connect(ctx context.Context, url string) (*nats.Conn, error) {
	var last error
	for attempt := 0; attempt < 40; attempt++ {
		nc, err := nats.Connect(url, nats.MaxReconnects(0), nats.Timeout(3*time.Second))
		if err == nil {
			return nc, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return nil, fmt.Errorf("natstest: connect %s after retries: %w", url, last)
}
