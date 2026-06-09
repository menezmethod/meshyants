// Package oracle provides LLM-backed Oracle Interface Agents.
package oracle

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/oracle/policy"
)

//go:embed intent_schema.json
var intentSchema []byte

// OpenAIAdapter implements Adapter using OpenAI's structured outputs API.
type OpenAIAdapter struct {
	Config     Config
	HTTPClient *http.Client
	APIKey     string
}

// NewOpenAIAdapter creates an adapter with the given config.
// API key is read from OPENAI_API_KEY if not set.
func NewOpenAIAdapter(cfg Config) *OpenAIAdapter {
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 30 * time.Second
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}
	ak := cfg.APIKey
	if ak == "" {
		ak = os.Getenv("OPENAI_API_KEY")
	}
	return &OpenAIAdapter{
		Config: cfg,
		HTTPClient: &http.Client{Timeout: cfg.RequestTimeout},
		APIKey:     ak,
	}
}

// Canonicalize calls OpenAI with structured outputs to extract a TaskIntentHeader.
func (a *OpenAIAdapter) Canonicalize(ctx context.Context, goal string) (*policy.TaskIntentHeader, error) {
	if a.APIKey == "" {
		return nil, errors.New("OPENAI_API_KEY not set")
	}

	type chatReq struct {
		Model    string                 `json:"model"`
		Messages []map[string]string    `json:"messages"`
		Schema   map[string]interface{} `json:"response_format"`
	}

	schema := map[string]interface{}{}
	if err := json.Unmarshal(intentSchema, &schema); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	payload := chatReq{
		Model: a.Config.Model,
		Messages: []map[string]string{
			{"role": "system", "content": TaskIntentCanonicalizeSystemPrompt},
			{"role": "user", "content": goal},
		},
		Schema: schema,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, errors.New("openai: no choices returned")
	}

	content := result.Choices[0].Message.Content

	// Try unresolvable signal first.
	var unresolvable struct {
		Unresolvable bool `json:"unresolvable"`
	}
	if json.Unmarshal([]byte(content), &unresolvable); unresolvable.Unresolvable {
		return nil, ErrUnresolvable
	}

	// Parse as TaskIntentHeader.
	var header policy.TaskIntentHeader
	if err := json.Unmarshal([]byte(content), &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	if err := policy.ValidateRequiredSafetyFields(&header); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnresolvable, err)
	}

	return &header, nil
}

// Summarize produces a natural-language digest of recent pheromone records.
func (a *OpenAIAdapter) Summarize(ctx context.Context, records []*meshyantsv1.PheromoneRecord) (string, error) {
	if a.APIKey == "" {
		return "", errors.New("OPENAI_API_KEY not set")
	}
	if len(records) == 0 {
		return "No recent swarm activity.", nil
	}

	type summaryReq struct {
		Model    string                 `json:"model"`
		Messages []map[string]any       `json:"messages"`
	}

	// Build a concise summary of records.
	type recordSummary struct {
		Kind   string `json:"kind"`
		Record string `json:"record_id"`
		Subject string `json:"subject"`
	}
	summaries := make([]recordSummary, len(records))
	for i, r := range records {
		summaries[i] = recordSummary{
			Kind:    r.GetKind().String(),
			Record:  r.GetRecordId(),
			Subject: r.GetSubject(),
		}
	}
	recordsJSON, _ := json.Marshal(summaries)

	systemMsg := `You are the MeshyAnts Oracle Interface Agent. Summarize the recent swarm pheromone activity in plain English. Focus on SAFE (success) and DANGER (warnings) patterns. Keep it under 200 words.`
	payload := summaryReq{
		Model: a.Config.Model,
		Messages: []map[string]any{
			{"role": "system", "content": systemMsg},
			{"role": "user", "content": fmt.Sprintf("Recent pheromones:\n%s", string(recordsJSON))},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", errors.New("openai: no choices returned")
	}
	return result.Choices[0].Message.Content, nil
}

