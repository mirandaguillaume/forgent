package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	openRouterBaseURL = "https://openrouter.ai/api/v1/chat/completions"
	openRouterModel   = "anthropic/claude-sonnet-4"
	openRouterMaxTok  = 8192
)

// OpenRouterProvider calls the OpenRouter API (OpenAI-compatible format).
type OpenRouterProvider struct {
	apiKey  string
	baseURL string
	model   string
}

type openRouterRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	Messages  []openRouterMessage `json:"messages"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []openRouterChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type openRouterChoice struct {
	Message openRouterMessage `json:"message"`
}

// Complete sends a prompt to the OpenRouter API and returns the text response.
func (p *OpenRouterProvider) Complete(prompt string) (string, error) {
	reqBody := openRouterRequest{
		Model:     p.model,
		MaxTokens: openRouterMaxTok,
		Messages: []openRouterMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseURL
	if url == "" {
		url = openRouterBaseURL
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result openRouterResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return result.Choices[0].Message.Content, nil
}

func registerOpenRouterProvider() {
	RegisterProvider("openrouter", func(apiKey string) Provider {
		return &OpenRouterProvider{
			apiKey:  apiKey,
			baseURL: openRouterBaseURL,
			model:   openRouterModel,
		}
	})
}

func init() {
	registerOpenRouterProvider()
}
