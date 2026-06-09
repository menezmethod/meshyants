// Package dtn implements a resumable chunk sync overlay (OFFER/REQUEST/DATA/ACK/RESET) per docs/v1/10-failure-oriented-design-audit.md Pattern 3.
package dtn

import (
	"errors"
	"fmt"
	"sort"
)

var (
	ErrMissingChunk  = errors.New("dtn: missing chunk")
	ErrChunkConflict = errors.New("dtn: conflicting chunk data for same index")
	ErrResetRequired = errors.New("dtn: reset required")
)

// MessageType identifies overlay control messages.
type MessageType uint8

const (
	MsgOffer MessageType = iota + 1
	MsgRequest
	MsgData
	MsgAck
	MsgReset
)

// ChunkPayload is application data for one chunk index.
type ChunkPayload struct {
	Index int
	Data  []byte
}

// Session tracks reassembly state for one object transfer.
type Session struct {
	TotalChunks int
	chunks      map[int][]byte
	acked       map[int]struct{}
}

// NewSession creates a session expecting totalChunks chunks (indices 0..total-1).
func NewSession(totalChunks int) *Session {
	return &Session{
		TotalChunks: totalChunks,
		chunks:      make(map[int][]byte),
		acked:       make(map[int]struct{}),
	}
}

// HandleData stores a chunk; duplicate index must match same bytes (audit: U6).
func (s *Session) HandleData(idx int, data []byte) error {
	if idx < 0 || idx >= s.TotalChunks {
		return fmt.Errorf("dtn: chunk index out of range")
	}
	if prev, ok := s.chunks[idx]; ok {
		if string(prev) != string(data) {
			return fmt.Errorf("%w: index %d", ErrChunkConflict, idx)
		}
		return nil
	}
	cp := append([]byte(nil), data...)
	s.chunks[idx] = cp
	return nil
}

// HandleAck records server ack for a chunk.
func (s *Session) HandleAck(idx int) {
	s.acked[idx] = struct{}{}
}

// Offer returns sorted list of chunk indices we currently hold.
func (s *Session) Offer() []int {
	out := make([]int, 0, len(s.chunks))
	for i := range s.chunks {
		out = append(out, i)
	}
	sort.Ints(out)
	return out
}

// RequestMissing returns indices not yet present.
func (s *Session) RequestMissing() []int {
	var out []int
	for i := 0; i < s.TotalChunks; i++ {
		if _, ok := s.chunks[i]; !ok {
			out = append(out, i)
		}
	}
	return out
}

// Complete returns assembled payload when all chunks received.
func (s *Session) Complete() ([]byte, error) {
	if len(s.chunks) != s.TotalChunks {
		return nil, ErrMissingChunk
	}
	var buf []byte
	for i := 0; i < s.TotalChunks; i++ {
		c, ok := s.chunks[i]
		if !ok {
			return nil, ErrMissingChunk
		}
		buf = append(buf, c...)
	}
	return buf, nil
}

// Reset clears session state after crash or uncertain durability.
func (s *Session) Reset(totalChunks int) {
	s.TotalChunks = totalChunks
	s.chunks = make(map[int][]byte)
	s.acked = make(map[int]struct{})
}
