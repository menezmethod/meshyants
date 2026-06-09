package validate_test

import (
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/validate"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func baseTaskAtom() *meshyantsv1.TaskAtom {
	return &meshyantsv1.TaskAtom{
		SchemaVersion:    1,
		TaskId:           "t1",
		IdempotencyKey:   "k1",
		Issuer:           "oracle",
		TrustDomain:      "td",
		CausalParents:    []string{"root"},
		RequirementsJson: `{"tier":"executor-basic"}`,
		Payload:          []byte("{}"),
		ExpiresAt:        timestamppb.New(time.Now().Add(time.Hour)),
		Signature:        []byte{1, 2, 3},
	}
}

func TestTaskAtom_DTNBudget_Undersize(t *testing.T) {
	t.Parallel()
	ta := baseTaskAtom()
	require.NoError(t, validate.TaskAtom(ta, true))
}

func TestTaskAtom_DTNBudget_Oversize(t *testing.T) {
	t.Parallel()
	ta := baseTaskAtom()
	ta.Payload = make([]byte, validate.DTNMaxTaskAtomWireBytes)
	require.Error(t, validate.TaskAtom(ta, true))
}

func TestTaskAtom_ContentRefFitsDTN(t *testing.T) {
	t.Parallel()
	ta := baseTaskAtom()
	ta.Payload = nil
	ta.ContentRef = "bafybeef"
	require.NoError(t, validate.TaskAtom(ta, true))
}

func TestJoinGrant_Expired(t *testing.T) {
	t.Parallel()
	g := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		GrantId:       "g",
		TrustDomain:   "td",
		Issuer:        "q",
		ExpiresAt:     timestamppb.New(time.Now().Add(-time.Hour)),
		Signature:     []byte{1},
	}
	require.Error(t, validate.JoinGrant(g))
}

func TestUpdateManifest_Valid(t *testing.T) {
	t.Parallel()
	u := &meshyantsv1.UpdateManifest{
		SchemaVersion: 1,
		ReleaseId:     "r1",
		TrustDomain:   "td",
		Targets:       []string{"executor-basic"},
		CompatWindow:  ">=1.0.0 <2.0.0",
		Hashes:        map[string][]byte{"agent": {1}},
		Issuer:        "queen",
		ValidUntil:    timestamppb.New(time.Now().Add(time.Hour)),
		Signature:     []byte{1},
	}
	require.NoError(t, validate.UpdateManifest(u))
	_ = proto.Size(u)
}

func TestPheromoneRecord_Kinds(t *testing.T) {
	t.Parallel()
	p := &meshyantsv1.PheromoneRecord{
		SchemaVersion: 1,
		RecordId:      "p1",
		Kind:          meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER,
		Subject:       "task-1",
		Strength:      1,
		Issuer:        "worker",
		TrustDomain:   "td",
		CausalParents: []string{"t1"},
		IssuedAt:      timestamppb.Now(),
		Signature:     []byte{1},
	}
	require.NoError(t, validate.PheromoneRecord(p))
}

func TestCapabilityAdvertisement(t *testing.T) {
	t.Parallel()
	c := &meshyantsv1.CapabilityAdvertisement{
		SchemaVersion:    1,
		DeviceId:         "d1",
		TrustDomain:      "td",
		RuntimeTier:      meshyantsv1.RuntimeTier_RUNTIME_TIER_RELAY,
		TransportProfile: "fast-wide",
		ValidUntil:       timestamppb.New(time.Now().Add(time.Hour)),
		Signature:        []byte{1},
	}
	require.NoError(t, validate.CapabilityAdvertisement(c))
}

func TestTaskAtom_MissingIdempotency(t *testing.T) {
	t.Parallel()
	ta := baseTaskAtom()
	ta.IdempotencyKey = ""
	require.Error(t, validate.TaskAtom(ta, false))
}

func TestUpdateManifest_EmptyTargets(t *testing.T) {
	t.Parallel()
	u := &meshyantsv1.UpdateManifest{
		SchemaVersion: 1,
		ReleaseId:     "r",
		TrustDomain:   "td",
		CompatWindow:  "x",
		Hashes:        map[string][]byte{"a": {1}},
		Issuer:        "i",
		ValidUntil:    timestamppb.New(time.Now().Add(time.Hour)),
		Signature:     []byte{1},
	}
	require.Error(t, validate.UpdateManifest(u))
}

func TestJoinGrant_MissingGrantID(t *testing.T) {
	t.Parallel()
	g := &meshyantsv1.JoinGrant{
		SchemaVersion: 1,
		TrustDomain:   "td",
		Issuer:        "q",
		ExpiresAt:     timestamppb.New(time.Now().Add(time.Hour)),
		Signature:     []byte{1},
	}
	require.Error(t, validate.JoinGrant(g))
}

