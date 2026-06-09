package tier_test

import (
	"context"
	"testing"

	"github.com/meshyants/meshyants/v1/internal/runtime/tier"
	"github.com/stretchr/testify/require"
)

func TestDetect_ReturnsAtLeastRelay(t *testing.T) {
	t.Parallel()
	tr := tier.Detect(context.Background(), tier.DefaultOptions())
	require.GreaterOrEqual(t, int(tr), 1)
}
