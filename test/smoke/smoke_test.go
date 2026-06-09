package smoke_test

import (
	"context"
	"testing"

	"github.com/meshyants/meshyants/v1/internal/runtime/tier"
	"github.com/stretchr/testify/require"
)

func TestSmoke_RuntimeTierDetect(t *testing.T) {
	tr := tier.Detect(context.Background(), tier.DefaultOptions())
	require.NotEmpty(t, tr.String())
}
