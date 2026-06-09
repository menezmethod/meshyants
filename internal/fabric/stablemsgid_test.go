package fabric_test

import (
	"testing"

	"github.com/meshyants/meshyants/v1/internal/fabric"
	"github.com/stretchr/testify/require"
)

func TestStableMsgID_Deterministic(t *testing.T) {
	t.Parallel()
	a := fabric.StableMsgID("idem")
	b := fabric.StableMsgID("idem")
	require.Equal(t, a, b)
	require.NotEqual(t, a, fabric.StableMsgID("other"))
}
