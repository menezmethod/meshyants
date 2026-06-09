package merge_test

import (
	"testing"

	"github.com/meshyants/meshyants/v1/internal/merge"
	"github.com/stretchr/testify/require"
)

// audit: U7 — merge family boundaries reject unsupported artifacts.
func TestMerge_U7_Allowed(t *testing.T) {
	t.Parallel()
	require.NoError(t, merge.AllowNaiveCRDT("pheromone_summary"))
}

func TestMerge_U7_Rejected(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, merge.AllowNaiveCRDT("website_bundle"), merge.ErrDisallowedFamily)
	require.ErrorIs(t, merge.AllowNaiveCRDT(""), merge.ErrDisallowedFamily)
}
