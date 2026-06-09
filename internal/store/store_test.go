package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStore_PutAndGet(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(tmp)
	require.NoError(t, err)

	data := []byte("hello mesh")
	mh, err := s.Put(context.Background(), data)
	require.NoError(t, err)
	require.Contains(t, mh, "1220") // sha2-256 multicodec prefix

	retrieved, err := s.Get(context.Background(), mh)
	require.NoError(t, err)
	require.Equal(t, data, retrieved)
}

func TestStore_IdempotentPut(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(tmp)
	require.NoError(t, err)

	data := []byte("same data")
	mh1, err := s.Put(context.Background(), data)
	require.NoError(t, err)
	mh2, err := s.Put(context.Background(), data)
	require.NoError(t, err)
	require.Equal(t, mh1, mh2)
}

func TestStore_GetNotFound(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(tmp)
	require.NoError(t, err)

	_, err = s.Get(context.Background(), "1220deadbeef")
	require.Error(t, err)
}

func TestStore_VerifyIntegrity(t *testing.T) {
	tmp := t.TempDir()
	s, err := New(tmp)
	require.NoError(t, err)

	data := []byte("correct data")
	mh, err := s.Put(context.Background(), data)
	require.NoError(t, err)

	// Verify read works.
	retrieved, err := s.Get(context.Background(), mh)
	require.NoError(t, err)
	require.Equal(t, data, retrieved)

	// Tamper: overwrite the file with wrong content at the correct path.
	shardDir := filepath.Join(tmp, mh[:4])
	os.MkdirAll(shardDir, 0700)
	realPath := filepath.Join(shardDir, mh)
	err = os.WriteFile(realPath, []byte("tampered"), 0600)
	require.NoError(t, err)

	// Get should now fail integrity check.
	_, err = s.Get(context.Background(), mh)
	require.Error(t, err)
	require.Contains(t, err.Error(), "hash mismatch")
}
