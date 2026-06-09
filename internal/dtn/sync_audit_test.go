package dtn_test

import (
	"testing"

	"github.com/meshyants/meshyants/v1/internal/dtn"
	"github.com/stretchr/testify/require"
)

// audit: U6 — duplicate chunk same bytes OK; conflict errors.
func TestDTN_U6_Chunks(t *testing.T) {
	t.Parallel()
	s := dtn.NewSession(3)
	require.NoError(t, s.HandleData(1, []byte("b")))
	require.NoError(t, s.HandleData(1, []byte("b")))
	require.ErrorIs(t, s.HandleData(1, []byte("c")), dtn.ErrChunkConflict)

	require.NoError(t, s.HandleData(0, []byte("a")))
	require.NoError(t, s.HandleData(2, []byte("c")))
	buf, err := s.Complete()
	require.NoError(t, err)
	require.Equal(t, "abc", string(buf))
}

func TestDTN_U6_Reset(t *testing.T) {
	t.Parallel()
	s := dtn.NewSession(2)
	require.NoError(t, s.HandleData(0, []byte("a")))
	s.Reset(2)
	_, err := s.Complete()
	require.ErrorIs(t, err, dtn.ErrMissingChunk)
}
