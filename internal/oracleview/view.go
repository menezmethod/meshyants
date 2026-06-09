// Package oracleview formats blackboard records for human-facing surfaces without softening DANGER (audit: C8).
package oracleview

import (
	"encoding/json"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"google.golang.org/protobuf/encoding/protojson"
)

// RecordView is JSON-safe operator output preserving raw contract bytes.
type RecordView struct {
	Kind       string `json:"kind"`
	TrustDomain string `json:"trust_domain"`
	RecordID   string `json:"record_id,omitempty"`
	Subject    string `json:"subject,omitempty"`
	VerbatimJSON string `json:"verbatim_json"`
}

// FormatPheromones returns one view per record; DANGER text is not paraphrased (full protobuf JSON).
func FormatPheromones(records []*meshyantsv1.PheromoneRecord) ([]RecordView, error) {
	mo := protojson.MarshalOptions{EmitUnpopulated: true}
	out := make([]RecordView, 0, len(records))
	for _, r := range records {
		raw, err := mo.Marshal(r)
		if err != nil {
			return nil, err
		}
		kind := r.GetKind().String()
		out = append(out, RecordView{
			Kind:         kind,
			TrustDomain:  r.GetTrustDomain(),
			RecordID:     r.GetRecordId(),
			Subject:      r.GetSubject(),
			VerbatimJSON: string(raw),
		})
	}
	return out, nil
}

// ToJSON encodes views for CLI/API.
func ToJSON(views []RecordView) ([]byte, error) {
	return json.MarshalIndent(views, "", "  ")
}
