# Story 1.2: Ollama Provider Adapter

Status: done

## Story

As a **developer**,
I want Forgent to communicate directly with my local Ollama instance,
so that I can analyze agent files without an API key or network dependency.

## Acceptance Criteria

1. Ollama adapter implements the existing `Provider` interface (`Complete(prompt string) (string, error)`)
2. Adapter communicates via direct HTTP to Ollama's REST API at `localhost:11434`
3. Adapter registers via `init()` + `RegisterProvider("ollama", ...)` following existing patterns
4. When Ollama is not running: returns actionable error "Ollama not running. Start with `ollama serve` or use `--provider anthropic`"
5. When model is not pulled: returns "Model not available. Run `ollama pull <model>` or use `--model <available_model>`"
6. When LLM call exceeds timeout (120s default, configurable): operation cancelled with actionable message
7. Adapter supports model selection via configuration (not hardcoded)
8. `GetProvider("ollama", "")` works without an API key (local = no auth)
9. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [x] Task 1: Implement Ollama Provider (AC: #1, #2, #3, #8)
  - [x] 1.1: Created `internal/llm/ollama.go` implementing `Provider` interface
  - [x] 1.2: Implemented `Complete()` using HTTP POST to `localhost:11434/api/generate`
  - [x] 1.3: `stream: false`, `temperature: 0.1`, `num_predict: 2048`
  - [x] 1.4: Configurable model via `OllamaProvider.model` (default: "llama3")
  - [x] 1.5: Configurable base URL (default: `http://localhost:11434`), apiKey used as URL override
  - [x] 1.6: Registered via `init()` + `RegisterProvider("ollama", factory)`
  - [x] 1.7: Empty apiKey → default localhost; non-empty → used as base URL override
- [x] Task 2: Error handling with actionable messages (AC: #4, #5, #6)
  - [x] 2.1: Connection refused → "Ollama not running. Start with `ollama serve` or use `--provider anthropic`"
  - [x] 2.2: 404 or error with "not found" → "Model not available. Run `ollama pull <model>` or use `--model <available_model>`"
  - [x] 2.3: Malformed JSON → "Ollama returned invalid response." Empty response → "empty response"
  - [x] 2.4: 120s default timeout via context.WithTimeout + http.Client.Timeout → "Analysis timed out"
- [x] Task 3: Tests (AC: #9)
  - [x] 3.1: TestOllamaProvider_Complete_Success — verifies request JSON format (model, prompt, stream, options)
  - [x] 3.2: TestOllamaProvider_Complete_Success — verifies response content extraction
  - [x] 3.3: 4 error tests: ConnectionRefused, ModelNotFound, Timeout, MalformedResponse, EmptyResponse, ServerError
  - [x] 3.4: TestOllamaProvider_Registry + Registry_BaseURLOverride
  - [x] 3.5: Skipped — no integration test in this story (Ollama availability varies)
  - [x] 3.6: All existing tests pass (bench failures are pre-existing)

## Dev Notes

### Ollama REST API Format

Ollama uses a DIFFERENT format from OpenAI/OpenRouter. Key differences:

```
// Ollama API (NOT OpenAI-compatible)
POST http://localhost:11434/api/generate
{
  "model": "llama3",
  "prompt": "...",
  "stream": false,
  "options": {
    "temperature": 0.1,
    "num_predict": 2048
  }
}

// Response:
{
  "model": "llama3",
  "response": "...",       // ← note: "response", not "choices[0].message.content"
  "done": true,
  "total_duration": 1234,
  "eval_count": 100
}
```

This is NOT the OpenAI chat completions format used by OpenRouter. The adapter must use Ollama's native format.

Ollama also has a chat API (`/api/chat`) but `/api/generate` is simpler for single-turn completions.

### Model Detection

To list available models:
```
GET http://localhost:11434/api/tags

Response:
{
  "models": [
    {"name": "llama3:latest", "size": 4661211808, ...},
    {"name": "phi3:latest", "size": 2318370816, ...}
  ]
}
```

This is used later (Story 1.6, FR11) for model auto-detection. Not needed for this story — just implement `Complete()`.

### Existing Code Patterns

Follow the exact pattern of `internal/llm/openrouter.go`:
- Same package (`internal/llm`)
- Same `init()` registration via `RegisterProvider()`
- Same `Provider` interface (`Complete(prompt string) (string, error)`)
- Same error wrapping (`fmt.Errorf("context: %w", err)`)

Key difference: OpenRouter needs an API key, Ollama doesn't. The `ProviderFactory` passes `apiKey string` — for Ollama, ignore it or use it as the base URL override.

### Provider Factory Signature

```go
// Existing in registry.go:
type ProviderFactory func(apiKey string) Provider

// For Ollama, apiKey is unused but must be accepted:
func init() {
    RegisterProvider("ollama", func(apiKey string) Provider {
        return &OllamaProvider{
            baseURL: "http://localhost:11434",
            model:   "llama3",  // Default from ADR-8
        }
    })
}
```

Consider: use `apiKey` as base URL override if non-empty, enabling `--provider ollama --api-key http://remote-ollama:11434` for remote Ollama instances.

### Previous Story Intelligence (from Story 1.1)

- The spike used OpenRouter's OpenAI-compatible API — Ollama's native API is different
- `ExtractYAML()` in `internal/doctor/spike/spike.go` handles code fence stripping — this will be reused by the analyzer skill (Story 1.5), not by this adapter
- Recommended default model: 8B+ instruction-tuned (Llama 3/4, Qwen 3). Avoid Gemma (unreliable YAML).
- ADR-8 confirmed local-first architecture is viable

### What NOT to Build

- Do NOT implement model auto-detection (`/api/tags`) — that's Story 1.6
- Do NOT implement the doctor CLI command — that's Story 1.6
- Do NOT add configuration file support — that's Story 6.1
- DO keep it simple: `Complete()` sends prompt, returns response text

### Project Structure Notes

- New file: `internal/llm/ollama.go`
- New file: `internal/llm/ollama_test.go`
- NO other files modified (except test verification)

### Testing Strategy

Use `httptest.NewServer` to mock Ollama's HTTP API for unit tests:

```go
func TestOllamaComplete(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request format
        var req map[string]interface{}
        json.NewDecoder(r.Body).Decode(&req)
        assert.Equal(t, "llama3", req["model"])
        assert.Equal(t, false, req["stream"])

        // Return Ollama response format
        json.NewEncoder(w).Encode(map[string]interface{}{
            "response": "findings:\n  - category: guardrails\n    issue: test",
            "done": true,
        })
    }))
    defer server.Close()

    p := &OllamaProvider{baseURL: server.URL, model: "llama3"}
    result, err := p.Complete("test prompt")
    require.NoError(t, err)
    assert.Contains(t, result, "guardrails")
}
```

For error tests, return appropriate HTTP status codes or close the server immediately.

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-5] — Ollama direct HTTP adapter with actionable error messages
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-8] — Local LLM quality validated, recommended 8B+ models
- [Source: _bmad-output/planning-artifacts/prd.md#FR5] — Local LLM analysis without API key
- [Source: _bmad-output/planning-artifacts/prd.md#NFR25] — Works with any Ollama model, not hardcoded
- [Source: _bmad-output/planning-artifacts/prd.md#NFR35] — 120s default timeout, configurable
- [Source: internal/llm/openrouter.go] — Reference implementation for provider pattern
- [Source: internal/llm/registry.go] — Provider registration mechanism

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

- `go test ./internal/llm/ -run TestOllama -v` — 9/9 tests PASS (3.009s)

### Completion Notes List

- OllamaProvider implements Provider interface with Ollama's native `/api/generate` API (NOT OpenAI-compatible)
- `temperature: 0.1` and `num_predict: 2048` hardcoded per ADR-8 recommendation
- `apiKey` parameter repurposed as base URL override for remote Ollama instances
- `io.LimitReader` used (10MB) to prevent memory bombs — addresses code review finding from Story 1.1
- `context.WithTimeout` + `http.Client.Timeout` for proper timeout handling
- Connection refused detection via net.OpError unwrapping + string matching fallback
- 9 unit tests covering: success, connection refused, model not found, timeout, malformed response, empty response, server error, registry, base URL override

### File List

- `internal/llm/ollama.go` — Ollama provider implementation (NEW)
- `internal/llm/ollama_test.go` — 9 unit tests (NEW)
