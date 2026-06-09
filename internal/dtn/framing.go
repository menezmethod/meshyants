// Package dtn implements the delay-tolerant chunk sync overlay with OFFER/REQUEST/DATA/ACK/RESET
// message framing (docs/v1/10-failure-oriented-design-audit.md Pattern 3).
package dtn

import (
	"encoding/binary"
	"fmt"
)

// MsgType identifies the chunk protocol message type.
type MsgType uint8

const (
	MsgChunkOffer  MsgType = 1
	MsgChunkReq    MsgType = 2 // REQUEST
	MsgChunkData   MsgType = 3
	MsgChunkAck    MsgType = 4
	MsgChunkReset  MsgType = 5
)

// ChunkOffer advertises available chunk IDs.
type ChunkOffer struct {
	SessionID string
	ChunkIDs  []uint64
}

// ChunkReq requests specific chunk IDs.
type ChunkReq struct {
	SessionID string
	ChunkIDs  []uint64
}

// ChunkData transmits chunk bytes.
type ChunkData struct {
	SessionID string
	ChunkID   uint64
	Bytes     []byte
}

// ChunkAck acknowledges receipt of chunks.
type ChunkAck struct {
	SessionID string
	ChunkIDs  []uint64
}

// ChunkReset aborts a session.
type ChunkReset struct {
	SessionID string
	Reason    string
}

// Encode encodes a message for transport over JetStream or other carriers.
func Encode(msg any) ([]byte, error) {
	switch m := msg.(type) {
	case ChunkOffer:
		return encodeOffer(m), nil
	case ChunkReq:
		return encodeReq(m), nil
	case ChunkData:
		return encodeData(m), nil
	case ChunkAck:
		return encodeAck(m), nil
	case ChunkReset:
		return encodeReset(m), nil
	default:
		return nil, fmt.Errorf("dtn: unknown message type %T", msg)
	}
}

func encodeOffer(m ChunkOffer) []byte {
	return encodeChunks(MsgChunkOffer, m.SessionID, m.ChunkIDs, nil)
}

func encodeReq(m ChunkReq) []byte {
	return encodeChunks(MsgChunkReq, m.SessionID, m.ChunkIDs, nil)
}

func encodeAck(m ChunkAck) []byte {
	return encodeChunks(MsgChunkAck, m.SessionID, m.ChunkIDs, nil)
}

func encodeChunks(t MsgType, sessionID string, chunkIDs []uint64, _ []byte) []byte {
	// Layout: msgType(1) + sessionLen(2) + sessionID + chunkCount(2) + [chunkIDs...]
	size := 1 + 2 + len(sessionID) + 2 + len(chunkIDs)*8
	buf := make([]byte, size)
	buf[0] = byte(t)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(sessionID)))
	copy(buf[3:], sessionID)
	offset := 3 + len(sessionID)
	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(len(chunkIDs)))
	offset += 2
	for i, id := range chunkIDs {
		binary.BigEndian.PutUint64(buf[offset+i*8:offset+i*8+8], id)
	}
	return buf
}

func encodeData(m ChunkData) []byte {
	// Layout: msgType(1) + sessionLen(2) + sessionID + chunkID(8) + bytes
	size := 1 + 2 + len(m.SessionID) + 8 + len(m.Bytes)
	buf := make([]byte, size)
	buf[0] = byte(MsgChunkData)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(m.SessionID)))
	copy(buf[3:], m.SessionID)
	offset := 3 + len(m.SessionID)
	binary.BigEndian.PutUint64(buf[offset:offset+8], m.ChunkID)
	copy(buf[offset+8:], m.Bytes)
	return buf
}

func encodeReset(m ChunkReset) []byte {
	// Layout: msgType(1) + sessionLen(2) + sessionID + reasonLen(2) + reason
	size := 1 + 2 + len(m.SessionID) + 2 + len(m.Reason)
	buf := make([]byte, size)
	buf[0] = byte(MsgChunkReset)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(m.SessionID)))
	copy(buf[3:], m.SessionID)
	offset := 3 + len(m.SessionID)
	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(len(m.Reason)))
	copy(buf[offset+2:], m.Reason)
	return buf
}

// Decode parses a transport message back into its structured form.
func Decode(buf []byte) (any, error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("dtn: message too short")
	}
	t := MsgType(buf[0])
	slen := int(binary.BigEndian.Uint16(buf[1:3]))
	if 3+slen > len(buf) {
		return nil, fmt.Errorf("dtn: truncated session ID")
	}
	sessionID := string(buf[3 : 3+slen])
	offset := 3 + slen

	switch t {
	case MsgChunkOffer, MsgChunkReq, MsgChunkAck:
		if offset+2 > len(buf) {
			return nil, fmt.Errorf("dtn: truncated chunk count")
		}
		count := int(binary.BigEndian.Uint16(buf[offset:offset+2]))
		offset += 2
		if offset+count*8 > len(buf) {
			return nil, fmt.Errorf("dtn: truncated chunk list")
		}
		ids := make([]uint64, count)
		for i := 0; i < count; i++ {
			ids[i] = binary.BigEndian.Uint64(buf[offset+i*8:offset+i*8+8])
		}
		switch t {
		case MsgChunkOffer:
			return ChunkOffer{SessionID: sessionID, ChunkIDs: ids}, nil
		case MsgChunkReq:
			return ChunkReq{SessionID: sessionID, ChunkIDs: ids}, nil
		case MsgChunkAck:
			return ChunkAck{SessionID: sessionID, ChunkIDs: ids}, nil
		}
	case MsgChunkData:
		if offset+8 > len(buf) {
			return nil, fmt.Errorf("dtn: truncated chunk data header")
		}
		chunkID := binary.BigEndian.Uint64(buf[offset : offset+8])
		offset += 8
		return ChunkData{SessionID: sessionID, ChunkID: chunkID, Bytes: buf[offset:]}, nil
	case MsgChunkReset:
		if offset+2 > len(buf) {
			return nil, fmt.Errorf("dtn: truncated reset")
		}
		rlen := int(binary.BigEndian.Uint16(buf[offset:offset+2]))
		offset += 2
		if offset+rlen > len(buf) {
			return nil, fmt.Errorf("dtn: truncated reset reason")
		}
		return ChunkReset{SessionID: sessionID, Reason: string(buf[offset:offset+rlen])}, nil
	}
	return nil, fmt.Errorf("dtn: unknown message type %d", t)
}
