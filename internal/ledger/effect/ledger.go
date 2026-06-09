// Package effect is a consumer-side effect ledger for idempotent side effects (audit: U5, I4, I5).
package effect

import (
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketName  = []byte("effect_ledger")
	ErrRecorded = errors.New("effect: already recorded")
)

// Ledger tracks applied side effects by stable key (e.g. idempotency_key).
type Ledger struct {
	db *bolt.DB
}

// Open opens or creates the ledger file at path.
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

func (l *Ledger) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}

// TryApply returns (true, nil) if this is the first time key is seen and records it.
// Returns (false, nil) if key was already applied (replay / duplicate delivery).
func (l *Ledger) TryApply(effectKey string) (applied bool, err error) {
	err = l.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		k := []byte(effectKey)
		if b.Get(k) != nil {
			applied = false
			return nil
		}
		if err := b.Put(k, []byte{1}); err != nil {
			return err
		}
		applied = true
		return nil
	})
	return applied, err
}

// Has returns whether the effect key was recorded.
func (l *Ledger) Has(effectKey string) (bool, error) {
	var ok bool
	err := l.db.View(func(tx *bolt.Tx) error {
		ok = tx.Bucket(bucketName).Get([]byte(effectKey)) != nil
		return nil
	})
	return ok, err
}
