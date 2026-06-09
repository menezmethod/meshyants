// Package signing provides Ed25519 sign/verify for protobuf contracts with a dedicated signature field.
package signing

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	ErrInvalidSignature = errors.New("signing: invalid signature")
	ErrNoSignatureField = errors.New("signing: message has no signature field")
)

// MarshalCanonical returns the wire encoding of m with the signature field cleared, for signing or verification inputs.
func MarshalCanonical(m proto.Message) ([]byte, error) {
	cloned := proto.Clone(m)
	r := cloned.ProtoReflect()
	fd := r.Descriptor().Fields().ByName("signature")
	if fd == nil {
		return nil, ErrNoSignatureField
	}
	r.Clear(fd)
	return proto.Marshal(cloned)
}

// Sign sets the signature field on m using priv. The signed bytes exclude the signature field.
func Sign(m proto.Message, priv ed25519.PrivateKey) error {
	payload, err := MarshalCanonical(m)
	if err != nil {
		return err
	}
	sig := ed25519.Sign(priv, payload)
	r := m.ProtoReflect()
	fd := r.Descriptor().Fields().ByName("signature")
	if fd == nil {
		return ErrNoSignatureField
	}
	r.Set(fd, protoreflect.ValueOfBytes(sig))
	return nil
}

// Verify checks the signature field on m using pub.
func Verify(m proto.Message, pub ed25519.PublicKey) error {
	payload, err := MarshalCanonical(m)
	if err != nil {
		return err
	}
	r := m.ProtoReflect()
	fd := r.Descriptor().Fields().ByName("signature")
	if fd == nil {
		return ErrNoSignatureField
	}
	sig := r.Get(fd).Bytes()
	if len(sig) == 0 {
		return fmt.Errorf("%w: empty signature", ErrInvalidSignature)
	}
	if !ed25519.Verify(pub, payload, sig) {
		return ErrInvalidSignature
	}
	return nil
}

// GenerateKeyPair returns a new Ed25519 key pair for tests and dev trust domains.
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(nil)
}

// LoadPrivateKeyFromFile reads a 32-byte raw Ed25519 seed or standard base64 encoding of those bytes.
func LoadPrivateKeyFromFile(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("signing: read key file: %w", err)
	}
	dec := make([]byte, 32)
	n, err := base64.StdEncoding.Decode(dec, data)
	if err != nil {
		if len(data) == 32 {
			copy(dec, data)
			n = 32
		} else {
			return nil, fmt.Errorf("signing: key must be 32 bytes (raw) or base64")
		}
	}
	if n != 32 {
		return nil, fmt.Errorf("signing: expected 32 key bytes, got %d", n)
	}
	// PrivateKey on the wire is often a 32-byte seed; Go's Sign API expects NewKeyFromSeed.
	return ed25519.NewKeyFromSeed(dec[:32]), nil
}
