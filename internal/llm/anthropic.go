package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL           = "https://api.anthropic.com/v1/messages"
	anthropicVersion         = "2023-06-01"
	anthropicDefaultModel    = "claude-sonnet-4-20250514"
	anthropicMaxTok          = 8192
	anthropicDefaultTimeout  = 120 * time.Second
)

// AnthropicProvider calls the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey  string
	model   string
	baseURL string
	timeout time.Duration
}

// resolveAnthropicModel maps short model names to full API model IDs.
func resolveAnthropicModel(name string) string {
	aliases := map[string]string{
		"haiku":   "claude-haiku-4-5-20251001",
		"sonnet":  "claude-sonnet-4-20250514",
		"opus":    "claude-opus-4-20250514",
	}
	if full, ok := aliases[name]; ok {
		return full
	}
	return name // assume it's already a full model ID
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SetModel sets the model to use for API calls. Accepts short aliases (haiku, sonnet, opus)
// or full model IDs (claude-haiku-4-5-20251001).
func (p *AnthropicProvider) SetModel(model string) {
	p.model = resolveAnthropicModel(model)
}

// Complete sends a prompt to the Anthropic Messages API and returns the text response.
func (p *AnthropicProvider) Complete(prompt string) (string, error) {
	model := p.model
	if model == "" {
		model = anthropicDefaultModel
	}

	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: anthropicMaxTok,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	timeout := p.timeout
	if timeout == 0 {
		timeout = anthropicDefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := p.baseURL
	if url == "" {
		url = defaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return "", fmt.Errorf("Anthropic request timed out after %s. Try a smaller file or use `--provider ollama`", timeout)
		}
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB limit
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return result.Content[0].Text, nil
}

func registerAnthropicProvider() {
	RegisterProvider("anthropic", func(apiKey string) Provider {
		return &AnthropicProvider{
			apiKey:  apiKey,
			baseURL: defaultBaseURL,
			timeout: anthropicDefaultTimeout,
		}
	})
}

func init() {
	registerAnthropicProvider()
}
