// Queen is the MeshyAnts provisioning authority. It issues JoinGrant and ProvisioningManifest
// signed contracts to bootstrapping nodes (docs/v1/02-installation-onboarding-and-governance.md).
//
// Production: set --key-file to a hardware key or HSM-backed key.
// Dev: run without --key-file to use an ephemeral key.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/signing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	addr       = flag.String("addr", ":8090", "HTTP listen address")
	keyFile    = flag.String("key-file", "", "Ed25519 private key file (32 bytes, base64 or raw binary)")
	issuerName = flag.String("issuer", "queen", "signing issuer name")
	td         = flag.String("trust-domain", "mesh", "default trust domain")
	keyTTL     = flag.Duration("key-ttl", 24*time.Hour, "JoinGrant/Manifest validity")
)

// KeyStore manages the queen's signing key.
type KeyStore struct{ key ed25519.PrivateKey }

func (s *KeyStore) Get() ed25519.PrivateKey { return s.key }

func loadKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	dec := make([]byte, 32)
	n, err := base64.StdEncoding.Decode(dec, data)
	if err != nil {
		if len(data) == 32 {
			copy(dec, data)
			n = 32
		} else {
			return nil, fmt.Errorf("key must be 32 bytes (raw) or base64")
		}
	}
	if n != 32 {
		return nil, fmt.Errorf("expected 32 bytes, got %d", n)
	}
	return ed25519.NewKeyFromSeed(dec[:32]), nil
}

// Queen is the provisioning HTTP service.
type Queen struct{ store *KeyStore }

// GrantRequest is the POST /grant body.
type GrantRequest struct {
	DeviceID    string `json:"device_id"`
	TrustDomain string `json:"trust_domain"`
}

// GrantResponse is the base64-encoded JoinGrant protobuf.
type GrantResponse struct{ JoinGrant string `json:"join_grant_base64"` }

// ProvisionRequest is the POST /provision/:deviceID body.
type ProvisionRequest struct {
	RuntimeTier meshyantsv1.RuntimeTier `json:"runtime_tier"`
	BuildPolicy string                   `json:"build_policy"`
}

// ProvisionResponse is the base64-encoded ProvisioningManifest protobuf.
type ProvisionResponse struct{ Manifest string `json:"manifest_base64"` }

// BundleRequest is the POST /bundle body (offline provisioning).
type BundleRequest struct {
	DeviceID    string                  `json:"device_id"`
	TrustDomain string                  `json:"trust_domain"`
	RuntimeTier meshyantsv1.RuntimeTier `json:"runtime_tier"`
	BuildPolicy string                   `json:"build_policy"`
}

func main() {
	flag.Parse()

	var priv ed25519.PrivateKey
	if *keyFile != "" {
		var err error
		priv, err = loadKey(*keyFile)
		if err != nil {
			log.Fatalf("load key: %v", err)
		}
	} else {
		log.Print("WARNING: ephemeral key (not for production)")
		_, p, err := ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatalf("generate key: %v", err)
		}
		priv = p
	}

	q := &Queen{store: &KeyStore{key: priv}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /grant", q.handleGrant)
	mux.HandleFunc("POST /provision/{deviceID}", q.handleProvision)
	mux.HandleFunc("GET /health", q.handleHealth)
	mux.HandleFunc("GET /public-key", q.handlePublicKey)
	mux.HandleFunc("POST /bundle", q.handleBundle)

	log.Printf("queen listening on %s (issuer=%s, td=%s)", *addr, *issuerName, *td)
	srv := &http.Server{Addr: *addr, Handler: mux, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func (q *Queen) defaultTD(reqTD string) string {
	if reqTD != "" {
		return reqTD
	}
	return *td
}

func (q *Queen) handleGrant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req GrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("decode: %v", err), http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "device_id required", http.StatusBadRequest)
		return
	}
	trustDomain := q.defaultTD(req.TrustDomain)
	g := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		GrantId:       fmt.Sprintf("grant-%s-%d", req.DeviceID, time.Now().Unix()),
		TrustDomain:   trustDomain,
		Issuer:        *issuerName,
		ExpiresAt:     timestamppb.New(time.Now().Add(*keyTTL)),
	}
	if err := signing.Sign(g, q.store.Get()); err != nil {
		http.Error(w, fmt.Sprintf("sign: %v", err), http.StatusInternalServerError)
		return
	}
	payload, err := proto.Marshal(g)
	if err != nil {
		http.Error(w, fmt.Sprintf("marshal: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(GrantResponse{JoinGrant: base64.StdEncoding.EncodeToString(payload)})
}

func (q *Queen) handleProvision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	deviceID := r.PathValue("deviceID")
	if deviceID == "" {
		http.Error(w, "deviceID required", http.StatusBadRequest)
		return
	}
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("decode: %v", err), http.StatusBadRequest)
		return
	}
	m := &meshyantsv1.ProvisioningManifest{
		SchemaVersion: 1,
		DeviceId:     deviceID,
		TrustDomain:  *td,
		RuntimeTier: req.RuntimeTier,
		Issuer:      *issuerName,
		BuildPolicy: req.BuildPolicy,
		ExpiresAt:   timestamppb.New(time.Now().Add(*keyTTL)),
	}
	if err := signing.Sign(m, q.store.Get()); err != nil {
		http.Error(w, fmt.Sprintf("sign: %v", err), http.StatusInternalServerError)
		return
	}
	payload, err := proto.Marshal(m)
	if err != nil {
		http.Error(w, fmt.Sprintf("marshal: %v", err), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ProvisionResponse{Manifest: base64.StdEncoding.EncodeToString(payload)})
}

func (q *Queen) handleBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req BundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("decode: %v", err), http.StatusBadRequest)
		return
	}
	trustDomain := q.defaultTD(req.TrustDomain)
	g := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		GrantId:       fmt.Sprintf("grant-%s-%d", req.DeviceID, time.Now().Unix()),
		TrustDomain:   trustDomain,
		Issuer:        *issuerName,
		ExpiresAt:     timestamppb.New(time.Now().Add(*keyTTL)),
	}
	if err := signing.Sign(g, q.store.Get()); err != nil {
		http.Error(w, fmt.Sprintf("sign grant: %v", err), http.StatusInternalServerError)
		return
	}
	m := &meshyantsv1.ProvisioningManifest{
		SchemaVersion: 1,
		DeviceId:     req.DeviceID,
		TrustDomain: trustDomain,
		RuntimeTier: req.RuntimeTier,
		Issuer:      *issuerName,
		BuildPolicy: req.BuildPolicy,
		ExpiresAt:   timestamppb.New(time.Now().Add(*keyTTL)),
	}
	if err := signing.Sign(m, q.store.Get()); err != nil {
		http.Error(w, fmt.Sprintf("sign manifest: %v", err), http.StatusInternalServerError)
		return
	}
	gB, _ := proto.Marshal(g)
	mB, _ := proto.Marshal(m)
	json.NewEncoder(w).Encode(map[string]string{
		"join_grant_b64":            base64.StdEncoding.EncodeToString(gB),
		"provisioning_manifest_b64": base64.StdEncoding.EncodeToString(mB),
	})
}

func (q *Queen) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "issuer": *issuerName, "td": *td})
}

func (q *Queen) handlePublicKey(w http.ResponseWriter, r *http.Request) {
	pub := q.store.Get().Public().(ed25519.PublicKey)
	json.NewEncoder(w).Encode(map[string]string{
		"public_key": base64.StdEncoding.EncodeToString(pub),
		"issuer":     *issuerName,
	})
}

// EnsureKeyFile generates a new key file at path if it does not exist.
// Call at startup to auto-provision a key.
func EnsureKeyFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if dir := filepath.Dir(path); dir != "" {
		_ = os.MkdirAll(dir, 0700)
	}
	_, privB64, err := signing.GenerateKeyPair()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(privB64), 0600)
}
