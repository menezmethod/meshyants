package oracleview_test

import (
	"strings"
	"testing"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/oracleview"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// audit: C8 — DANGER must appear verbatim in structured output (protobuf JSON includes kind).
func TestOracleView_C8_DangerVerbatim(t *testing.T) {
	t.Parallel()
	p := &meshyantsv1.PheromoneRecord{
		SchemaVersion: 1,
		RecordId:      "r1",
		Kind:          meshyantsv1.PheromoneKind_PHEROMONE_KIND_DANGER,
		Subject:       "disk full on node-7",
		Strength:      1,
		Issuer:        "worker",
		TrustDomain:   "td",
		CausalParents: []string{"t1"},
		IssuedAt:      timestamppb.New(time.Now().UTC()),
		Signature:     []byte{1, 2, 3},
	}
	views, err := oracleview.FormatPheromones([]*meshyantsv1.PheromoneRecord{p})
	require.NoError(t, err)
	require.Len(t, views, 1)
	require.Contains(t, views[0].VerbatimJSON, "DANGER")
	require.Contains(t, views[0].VerbatimJSON, "disk full on node-7")
	require.Contains(t, strings.ToUpper(views[0].Kind), "DANGER")
}
