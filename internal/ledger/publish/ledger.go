// Package publish implements a durable publish ledger (audit: U4, Pattern 2 in docs/v1/10-failure-oriented-design-audit.md).
package publish

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketName     = []byte("publish_ledger")
	ErrBadState    = errors.New("publish: invalid state transition")
	ErrNotFound    = errors.New("publish: entry not found")
	ErrResurrected = errors.New("publish: cannot resurrect acked entry")
)

// State is the publish lifecycle for one outbound message.
type State uint8

const (
	StatePending State = iota + 1
	StateAcked
)

type Entry struct {
	PublishID   string
	MsgID       string
	PayloadHash []byte
	State       State
	CreatedUnix int64
}

// Ledger persists publish intent before async broker send.
type Ledger struct {
	db *bolt.DB
}

// Open opens or creates a ledger at path.
func Open(path string) (*Ledger, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucketIfNotExists(bucketName)
		return e
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Ledger{db: db}, nil
}

// Close releases the database.
func (l *Ledger) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}

func key(publishID string) []byte {
	return []byte(publishID)
}

// encode: state(1) + created(8) + len msgID(2) + msgID + len hash(2) + hash
func encode(e *Entry) []byte {
	msg := []byte(e.MsgID)
	h := e.PayloadHash
	if len(h) > 65535 || len(msg) > 65535 {
		panic("publish: field too large")
	}
	buf := make([]byte, 1+8+2+len(msg)+2+len(h))
	buf[0] = byte(e.State)
	binary.BigEndian.PutUint64(buf[1:9], uint64(e.CreatedUnix))
	binary.BigEndian.PutUint16(buf[9:11], uint16(len(msg)))
	copy(buf[11:], msg)
	o := 11 + len(msg)
	binary.BigEndian.PutUint16(buf[o:o+2], uint16(len(h)))
	copy(buf[o+2:], h)
	return buf
}

func decode(b []byte) (*Entry, error) {
	if len(b) < 1+8+2 {
		return nil, fmt.Errorf("publish: corrupt entry")
	}
	st := State(b[0])
	if st != StatePending && st != StateAcked {
		return nil, fmt.Errorf("publish: unknown state %d", st)
	}
	created := int64(binary.BigEndian.Uint64(b[1:9]))
	mlen := int(binary.BigEndian.Uint16(b[9:11]))
	if len(b) < 11+mlen+2 {
		return nil, fmt.Errorf("publish: corrupt entry")
	}
	msgID := string(b[11 : 11+mlen])
	o := 11 + mlen
	hlen := int(binary.BigEndian.Uint16(b[o : o+2]))
	if len(b) < o+2+hlen {
		return nil, fmt.Errorf("publish: corrupt entry")
	}
	hash := append([]byte(nil), b[o+2:o+2+hlen]...)
	return &Entry{MsgID: msgID, PayloadHash: hash, State: st, CreatedUnix: created}, nil
}

// PutPending records intent before sending to the broker.
func (l *Ledger) PutPending(publishID, msgID string, payloadHash []byte) error {
	return l.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		k := key(publishID)
		if v := b.Get(k); v != nil {
			ent, err := decode(v)
			if err != nil {
				return err
			}
			if ent.State == StateAcked {
				return fmt.Errorf("%w: %q", ErrResurrected, publishID)
			}
		}
		e := &Entry{
			PublishID:   publishID,
			MsgID:       msgID,
			PayloadHash: payloadHash,
			State:       StatePending,
			CreatedUnix: time.Now().Unix(),
		}
		return b.Put(k, encode(e))
	})
}

// MarkAcked transitions pending -> acked only.
func (l *Ledger) MarkAcked(publishID string) error {
	return l.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		k := key(publishID)
		v := b.Get(k)
		if v == nil {
			return ErrNotFound
		}
		ent, err := decode(v)
		if err != nil {
			return err
		}
		if ent.State != StatePending {
			return fmt.Errorf("%w: expected pending", ErrBadState)
		}
		ent.State = StateAcked
		ent.PublishID = publishID
		return b.Put(k, encode(ent))
	})
}

// Get returns the entry if present.
func (l *Ledger) Get(publishID string) (*Entry, error) {
	var out *Entry
	err := l.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketName).Get(key(publishID))
		if v == nil {
			return ErrNotFound
		}
		ent, err := decode(v)
		if err != nil {
			return err
		}
		ent.PublishID = publishID
		out = ent
		return nil
	})
	return out, err
}
