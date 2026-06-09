// Package store implements a content-addressed object store for content_ref overflow payloads
// (docs/v1/03-protocols-and-execution.md: oversize human goals via signed header + content_ref).
package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Store is a simple content-addressed store using SHA-256 multihash.
type Store struct {
	root string
}

// New creates a store at the given root directory.
func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, fmt.Errorf("store: mkdir: %w", err)
	}
	return &Store{root: root}, nil
}

// Put stores data and returns its multihash (sha256-256 identifier).
func (s *Store) Put(ctx context.Context, data []byte) (string, error) {
	_ = ctx
	hash := sha256.Sum256(data)
	mh := "1220" + hex.EncodeToString(hash[:]) // sha2-256 multicodec (0x1220)
	dir := filepath.Join(s.root, mh[:4])        // sharded by first 4 chars
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("store: mkdir: %w", err)
	}
	path := filepath.Join(dir, mh)
	if _, err := os.Stat(path); err == nil {
		return mh, nil // already stored
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", fmt.Errorf("store: write: %w", err)
	}
	return mh, nil
}

// Get retrieves data by its multihash.
func (s *Store) Get(ctx context.Context, multihash string) ([]byte, error) {
	_ = ctx
	if len(multihash) < 4 {
		return nil, fmt.Errorf("store: invalid multihash %q", multihash)
	}
	path := filepath.Join(s.root, multihash[:4], multihash)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("store: read: %w", err)
	}
	// Verify integrity.
	hash := sha256.Sum256(data)
	expected := "1220" + hex.EncodeToString(hash[:])
	if expected != multihash {
		return nil, fmt.Errorf("store: hash mismatch (corruption)")
	}
	return data, nil
}

// FileStore wraps a directory-backed store as an io.Writer.
type FileStore struct{ s *Store }

// NewFileStore wraps Store as an io.Writer (stores all written data).
func NewFileStore(s *Store) *FileStore { return &FileStore{s: s} }

// Write implements io.Writer, storing data and returning the multihash.
func (f *FileStore) Write(ctx context.Context, data []byte) (int, string, error) {
	mh, err := f.s.Put(ctx, data)
	return len(data), mh, err
}

// GetReader returns an io.Reader for the content at multihash.
func (s *Store) GetReader(ctx context.Context, multihash string) (io.Reader, error) {
	data, err := s.Get(ctx, multihash)
	if err != nil {
		return nil, err
	}
	return struct{ io.Reader }{io.NopCloser(io.LimitReader(nil, int64(len(data))))}, nil
}
