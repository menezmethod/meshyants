package oracle_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meshyants/meshyants/v1/internal/oracle"
	"github.com/stretchr/testify/require"
)

func TestMiniMaxAdapter_Canonicalize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var reqBody map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		require.Equal(t, true, reqBody["reasoning_split"])

		out := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{
					"content": `{"scope":"deploy","prohibited_actions":[],"deadline_rfc3339":"2027-01-01T00:00:00Z","transport_class":"fast-local","required_approvals":[],"canonical_goal":"deploy app to staging"}`,
				}},
			},
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("MINIMAX_BASE_URL", srv.URL+"/v1")
	a := oracle.NewMiniMaxAdapter(oracle.Config{
		APIKey:         "test-key",
		Model:          "MiniMax-M2.7",
		RequestTimeout: 5 * time.Second,
		MaxTokens:      256,
	})

	h, err := a.Canonicalize(context.Background(), "deploy to staging")
	require.NoError(t, err)
	require.Equal(t, "deploy", h.Scope)
	require.Equal(t, "fast-local", string(h.TransportClass))
}

func TestMiniMaxAdapter_Canonicalize_markdownFence(t *testing.T) {
	jsonLine := `{"scope":"x","prohibited_actions":[],"deadline_rfc3339":"2027-01-01T00:00:00Z","transport_class":"fast-wide","required_approvals":[],"canonical_goal":"g"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]interface{}{
					"content": "```json\n" + jsonLine + "\n```",
				}},
			},
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("MINIMAX_BASE_URL", srv.URL+"/v1")
	a := oracle.NewMiniMaxAdapter(oracle.Config{
		APIKey:         "k",
		RequestTimeout: 5 * time.Second,
	})
	h, err := a.Canonicalize(context.Background(), "goal")
	require.NoError(t, err)
	require.Equal(t, "x", h.Scope)
	require.Equal(t, "fast-wide", string(h.TransportClass))
}
