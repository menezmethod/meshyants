package oracle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	meshyantsv1 "github.com/meshyants/meshyants/v1/gen/meshyantsv1"
	"github.com/meshyants/meshyants/v1/internal/oracle/policy"
)

// MiniMaxAdapter calls MiniMax via the OpenAI-compatible HTTP API (Token Plan key).
// Docs: https://platform.minimax.io/docs/api-reference/text-openai-api
//
// Environment:
//   - MINIMAX_API_KEY (required): Token Plan or compatible API key
//   - MINIMAX_BASE_URL (optional): default https://api.minimax.io/v1
//   - MINIMAX_MODEL (optional): default MiniMax-M2.7
type MiniMaxAdapter struct {
	Config     Config
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
}

// NewMiniMaxAdapter builds an adapter. API key from cfg.APIKey or MINIMAX_API_KEY.
func NewMiniMaxAdapter(cfg Config) *MiniMaxAdapter {
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 60 * time.Second
	}
	if cfg.Model == "" {
		cfg.Model = "MiniMax-M2.7"
	}
	ak := strings.TrimSpace(cfg.APIKey)
	if ak == "" {
		ak = strings.TrimSpace(os.Getenv("MINIMAX_API_KEY"))
	}
	base := os.Getenv("MINIMAX_BASE_URL")
	if base == "" {
		base = "https://api.minimax.io/v1"
	}
	base = strings.TrimSuffix(base, "/")
	return &MiniMaxAdapter{
		Config:     cfg,
		HTTPClient: &http.Client{Timeout: cfg.RequestTimeout},
		APIKey:     ak,
		BaseURL:    base,
	}
}

func (a *MiniMaxAdapter) chatURL() string {
	return a.BaseURL + "/chat/completions"
}

// minimaxChatPayload returns the base fields PicoClaw-style OpenAI-compat clients use for M2:
// reasoning_split asks the API to put chain-of-thought in reasoning_content instead of content when supported.
func minimaxChatPayload(model string, temperature float64, messages []map[string]string, extra map[string]interface{}) map[string]interface{} {
	p := map[string]interface{}{
		"model":            model,
		"temperature":      temperature,
		"messages":         messages,
		"reasoning_split":  true,
	}
	for k, v := range extra {
		p[k] = v
	}
	return p
}

// Canonicalize extracts TaskIntentHeader using MiniMax M2.7 (OpenAI-compatible chat).
func (a *MiniMaxAdapter) Canonicalize(ctx context.Context, goal string) (*policy.TaskIntentHeader, error) {
	if a.APIKey == "" {
		return nil, errors.New("MINIMAX_API_KEY not set")
	}

	var schemaRoot map[string]interface{}
	if err := json.Unmarshal(intentSchema, &schemaRoot); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	payload := minimaxChatPayload(a.Config.Model, 1.0, []map[string]string{
		{"role": "system", "content": TaskIntentCanonicalizeSystemPrompt},
		{"role": "user", "content": goal},
	}, map[string]interface{}{
		"response_format": map[string]interface{}{
			"type": "json_schema",
			"json_schema": map[string]interface{}{
				"name":   "TaskIntentHeader",
				"strict": true,
				"schema": schemaRoot,
			},
		},
	})
	if a.Config.MaxTokens > 0 {
		payload["max_tokens"] = a.Config.MaxTokens
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.chatURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("minimax request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("minimax status %d: %s", resp.StatusCode, truncateForErr(respBody, 512))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, errors.New("minimax: no choices returned")
	}

	content := normalizeMiniMaxJSONPayload(result.Choices[0].Message.Content)

	var unresolvable struct {
		Unresolvable bool `json:"unresolvable"`
	}
	if json.Unmarshal([]byte(content), &unresolvable) == nil && unresolvable.Unresolvable {
		return nil, ErrUnresolvable
	}

	var header policy.TaskIntentHeader
	if err := json.Unmarshal([]byte(content), &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	if err := policy.ValidateRequiredSafetyFields(&header); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnresolvable, err)
	}
	return &header, nil
}

// Summarize turns recent pheromones into a short natural-language digest.
func (a *MiniMaxAdapter) Summarize(ctx context.Context, records []*meshyantsv1.PheromoneRecord) (string, error) {
	if a.APIKey == "" {
		return "", errors.New("MINIMAX_API_KEY not set")
	}
	if len(records) == 0 {
		return "No recent swarm activity.", nil
	}

	type recordSummary struct {
		Kind    string `json:"kind"`
		Record  string `json:"record_id"`
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

	systemMsg := `You are the MeshyAnts Oracle Interface Agent. Summarize the recent swarm pheromone activity in plain English. Focus on SAFE (success) and DANGER (warnings). Under 200 words.`

	payload := minimaxChatPayload(a.Config.Model, 1.0, []map[string]string{
		{"role": "system", "content": systemMsg},
		{"role": "user", "content": fmt.Sprintf("Recent pheromones:\n%s", string(recordsJSON))},
	}, nil)
	if a.Config.MaxTokens > 0 {
		payload["max_tokens"] = a.Config.MaxTokens
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.chatURL(), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("minimax request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("minimax status %d: %s", resp.StatusCode, truncateForErr(respBody, 512))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", errors.New("minimax: no choices returned")
	}
	out := normalizeMiniMaxPlainText(result.Choices[0].Message.Content)
	return out, nil
}

func truncateForErr(b []byte, max int) string {
	s := string(b)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// normalizeMiniMaxJSONPayload prepares message.content for json.Unmarshal.
func normalizeMiniMaxJSONPayload(s string) string {
	s = strings.TrimSpace(s)
	s = stripMarkdownCodeFence(s)
	s = strings.TrimSpace(s)
	s = stripThinkTags(s)
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") {
		return s
	}
	if obj := extractJSONObject(s); obj != "" {
		return obj
	}
	return s
}

func normalizeMiniMaxPlainText(s string) string {
	s = strings.TrimSpace(s)
	s = stripMarkdownCodeFence(s)
	s = strings.TrimSpace(s)
	s = stripThinkTags(s)
	return strings.TrimSpace(s)
}

func stripMarkdownCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimLeft(s, "\r\n\t ")
	if nl := strings.IndexByte(s, '\n'); nl != -1 {
		first := strings.TrimSpace(s[:nl])
		if first != "" && !strings.HasPrefix(first, "{") && !strings.HasPrefix(first, "[") {
			s = s[nl+1:]
		}
	}
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func extractJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func stripThinkTags(s string) string {
	pairs := []struct{ open, close string }{
		{"<think>", "</think>"},
		{"<thinking>", "</thinking>"},
		{"<reasoning>", "</reasoning>"},
	}
	for _, p := range pairs {
		for {
			start := strings.Index(s, p.open)
			if start == -1 {
				break
			}
			end := strings.Index(s, p.close)
			if end == -1 || end < start {
				break
			}
			s = s[:start] + s[end+len(p.close):]
		}
	}
	return strings.TrimSpace(s)
}
