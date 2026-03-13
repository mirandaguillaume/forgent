# `forgent import` Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `forgent import` command that converts existing agent markdown files and Vercel registry skills into composable Forgent YAML specs using LLM-assisted decomposition, validated by Forgent's existing lint/doctor/score tools.

**Architecture:** Pipeline-based — resolve source → extract frontmatter → LLM decomposition → validate (lint+doctor+score) → retry if low quality → preview → write. LLM provider is pluggable via interface. All validation reuses existing `internal/linter` and `internal/analyzer` packages.

**Tech Stack:** Go 1.22+, Cobra CLI, gopkg.in/yaml.v3, net/http (Anthropic API), testify, fatih/color

---

### Task 1: LLM Provider Interface + Mock

**Files:**
- Create: `internal/llm/provider.go`
- Create: `internal/llm/provider_test.go`

**Step 1: Write the failing test**

```go
package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Complete(prompt string) (string, error) {
	return m.response, m.err
}

func TestMockProviderImplementsInterface(t *testing.T) {
	var p Provider = &mockProvider{response: "hello"}
	result, err := p.Complete("test")
	assert.NoError(t, err)
	assert.Equal(t, "hello", result)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/llm/ -run TestMockProvider -v`
Expected: FAIL — package does not exist

**Step 3: Write minimal implementation**

```go
package llm

// Provider abstracts an LLM API for text completion.
type Provider interface {
	Complete(prompt string) (string, error)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/llm/ -run TestMockProvider -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/llm/provider.go internal/llm/provider_test.go
git commit -m "feat(import): add LLM provider interface"
```

---

### Task 2: Provider Registry

**Files:**
- Create: `internal/llm/registry.go`
- Create: `internal/llm/registry_test.go`

**Step 1: Write the failing test**

```go
package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAndGet(t *testing.T) {
	// Reset for test isolation
	providers = make(map[string]ProviderFactory)

	RegisterProvider("mock", func(apiKey string) Provider {
		return &mockProvider{response: "ok"}
	})

	p, err := GetProvider("mock", "key123")
	require.NoError(t, err)

	result, err := p.Complete("test")
	assert.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestGetUnregisteredProvider(t *testing.T) {
	providers = make(map[string]ProviderFactory)

	_, err := GetProvider("unknown", "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/llm/ -run TestRegister -v`
Expected: FAIL — undefined: ProviderFactory, RegisterProvider, GetProvider

**Step 3: Write minimal implementation**

```go
package llm

import "fmt"

// ProviderFactory creates a Provider given an API key.
type ProviderFactory func(apiKey string) Provider

var providers = make(map[string]ProviderFactory)

// RegisterProvider registers a named LLM provider factory.
func RegisterProvider(name string, factory ProviderFactory) {
	providers[name] = factory
}

// GetProvider returns a Provider by name, initialized with the given API key.
func GetProvider(name, apiKey string) (Provider, error) {
	factory, ok := providers[name]
	if !ok {
		available := make([]string, 0, len(providers))
		for k := range providers {
			available = append(available, k)
		}
		return nil, fmt.Errorf("unknown provider %q (available: %v)", name, available)
	}
	return factory(apiKey), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/llm/ -run TestRegister -v && go test ./internal/llm/ -run TestGetUnregistered -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/llm/registry.go internal/llm/registry_test.go
git commit -m "feat(import): add LLM provider registry"
```

---

### Task 3: Anthropic Provider

**Files:**
- Create: `internal/llm/anthropic.go`
- Create: `internal/llm/anthropic_test.go`

**Step 1: Write the failing test**

Test the request construction (not the actual API call — use httptest):

```go
package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		assert.Equal(t, "claude-sonnet-4-20250514", req["model"])

		// Return mock response
		resp := map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": `{"skills": [{"yaml": "skill: test\nversion: 0.1.0"}]}`},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &AnthropicProvider{
		apiKey:  "test-key",
		baseURL: server.URL,
	}

	result, err := p.Complete("analyze this agent")
	require.NoError(t, err)
	assert.Contains(t, result, "skills")
}

func TestAnthropicProviderRegistered(t *testing.T) {
	providers = make(map[string]ProviderFactory)
	registerAnthropicProvider()

	_, err := GetProvider("anthropic", "test-key")
	assert.NoError(t, err)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/llm/ -run TestAnthropic -v`
Expected: FAIL — undefined: AnthropicProvider, registerAnthropicProvider

**Step 3: Write implementation**

```go
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultAnthropicURL   = "https://api.anthropic.com/v1/messages"
	defaultAnthropicModel = "claude-sonnet-4-20250514"
	anthropicVersion      = "2023-06-01"
)

// AnthropicProvider calls the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
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
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *AnthropicProvider) Complete(prompt string) (string, error) {
	url := p.baseURL
	if url == "" {
		url = defaultAnthropicURL
	}

	reqBody := anthropicRequest{
		Model:     defaultAnthropicModel,
		MaxTokens: 8192,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
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
		return &AnthropicProvider{apiKey: apiKey}
	})
}

func init() {
	registerAnthropicProvider()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/llm/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/llm/anthropic.go internal/llm/anthropic_test.go
git commit -m "feat(import): add Anthropic LLM provider"
```

---

### Task 4: Source Resolution

**Files:**
- Create: `internal/importer/source.go`
- Create: `internal/importer/source_test.go`

**Step 1: Write the failing test**

```go
package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSourceType(t *testing.T) {
	tests := []struct {
		input    string
		expected SourceType
	}{
		{"vercel:code-reviewer", SourceVercel},
		{".claude/agents/review.md", SourceLocalFile},
		{".claude/", SourceLocalDir},
		{"my-agent.md", SourceLocalFile},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetectSourceType(tt.input))
		})
	}
}

func TestResolveLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.md")
	os.WriteFile(path, []byte("# My Agent\nDoes stuff"), 0644)

	sources, err := ResolveSources(path)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, "agent.md", sources[0].Name)
	assert.Contains(t, sources[0].Content, "My Agent")
}

func TestResolveLocalDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "review.md"), []byte("# Review"), 0644)
	os.WriteFile(filepath.Join(dir, "lint.agent.md"), []byte("# Lint"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore"), 0644)

	sources, err := ResolveSources(dir)
	require.NoError(t, err)
	assert.Len(t, sources, 2) // Only .md files
}

func TestDetectFramework(t *testing.T) {
	assert.Equal(t, FrameworkClaude, DetectFramework(".claude/agents/review.md"))
	assert.Equal(t, FrameworkCopilot, DetectFramework(".github/agents/review.agent.md"))
	assert.Equal(t, FrameworkUnknown, DetectFramework("my-agent.md"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -run TestDetect -v`
Expected: FAIL — package does not exist

**Step 3: Write implementation**

```go
package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SourceType indicates where the import source comes from.
type SourceType int

const (
	SourceLocalFile SourceType = iota
	SourceLocalDir
	SourceVercel
)

// Framework indicates which agent framework the source was written for.
type Framework int

const (
	FrameworkUnknown Framework = iota
	FrameworkClaude
	FrameworkCopilot
)

// Source represents a resolved input source with its content.
type Source struct {
	Name      string
	Path      string
	Content   string
	Framework Framework
}

// DetectSourceType determines the source type from the input string.
func DetectSourceType(input string) SourceType {
	if strings.HasPrefix(input, "vercel:") {
		return SourceVercel
	}
	info, err := os.Stat(input)
	if err == nil && info.IsDir() {
		return SourceLocalDir
	}
	return SourceLocalFile
}

// DetectFramework infers the agent framework from the file path.
func DetectFramework(path string) Framework {
	normalized := filepath.ToSlash(path)
	if strings.Contains(normalized, ".claude/") || strings.Contains(normalized, ".claude\\") {
		return FrameworkClaude
	}
	if strings.Contains(normalized, ".github/") || strings.Contains(normalized, ".github\\") {
		return FrameworkCopilot
	}
	return FrameworkUnknown
}

// ResolveSources reads source files based on the input path or registry reference.
func ResolveSources(input string) ([]Source, error) {
	switch DetectSourceType(input) {
	case SourceLocalFile:
		return resolveLocalFile(input)
	case SourceLocalDir:
		return resolveLocalDir(input)
	case SourceVercel:
		return resolveVercel(input)
	default:
		return nil, fmt.Errorf("unsupported source: %s", input)
	}
}

func resolveLocalFile(path string) ([]Source, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return []Source{{
		Name:      filepath.Base(path),
		Path:      path,
		Content:   string(content),
		Framework: DetectFramework(path),
	}}, nil
}

func resolveLocalDir(dir string) ([]Source, error) {
	var sources []Source
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		sources = append(sources, Source{
			Name:      name,
			Path:      path,
			Content:   string(content),
			Framework: DetectFramework(path),
		})
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no .md files found in %s", dir)
	}
	return sources, nil
}

func resolveVercel(input string) ([]Source, error) {
	// TODO: implement Vercel registry fetch
	name := strings.TrimPrefix(input, "vercel:")
	return nil, fmt.Errorf("Vercel registry not yet implemented (skill: %s)", name)
}
```

**Step 4: Run tests**

Run: `go test ./internal/importer/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/importer/source.go internal/importer/source_test.go
git commit -m "feat(import): add source resolution (local file, dir, vercel stub)"
```

---

### Task 5: Frontmatter Extraction

**Files:**
- Create: `internal/importer/frontmatter.go`
- Create: `internal/importer/frontmatter_test.go`

**Step 1: Write the failing test**

```go
package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFrontmatter(t *testing.T) {
	content := `---
name: ci-reviewer
description: Reviews code changes
tools: [Read, Write, Bash, Grep]
model: claude-sonnet-4-20250514
---

# CI Reviewer

You are a code review agent...`

	fm, body, err := ExtractFrontmatter(content)
	require.NoError(t, err)
	assert.Equal(t, "ci-reviewer", fm.Name)
	assert.Equal(t, "Reviews code changes", fm.Description)
	assert.Equal(t, []string{"Read", "Write", "Bash", "Grep"}, fm.Tools)
	assert.Contains(t, body, "CI Reviewer")
	assert.NotContains(t, body, "---")
}

func TestExtractFrontmatterNoFrontmatter(t *testing.T) {
	content := `# Just a markdown file

No frontmatter here.`

	fm, body, err := ExtractFrontmatter(content)
	require.NoError(t, err)
	assert.Empty(t, fm.Name)
	assert.Contains(t, body, "Just a markdown file")
}

func TestExtractFrontmatterToolsAsString(t *testing.T) {
	content := `---
tools: Read, Write, Bash
---
Body`

	fm, _, err := ExtractFrontmatter(content)
	require.NoError(t, err)
	assert.Equal(t, []string{"Read", "Write", "Bash"}, fm.Tools)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -run TestExtractFrontmatter -v`
Expected: FAIL — undefined: ExtractFrontmatter

**Step 3: Write implementation**

```go
package importer

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentFrontmatter holds parsed YAML frontmatter from an agent markdown file.
type AgentFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tools       []string `yaml:"-"`
	RawTools    interface{} `yaml:"tools"`
	Model       string   `yaml:"model"`
}

// ExtractFrontmatter parses YAML frontmatter from markdown content.
// Returns the parsed frontmatter, the markdown body (without frontmatter), and any error.
func ExtractFrontmatter(content string) (AgentFrontmatter, string, error) {
	var fm AgentFrontmatter

	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return fm, content, nil
	}

	// Find closing ---
	rest := trimmed[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return fm, content, nil
	}

	fmContent := rest[:idx]
	body := strings.TrimSpace(rest[idx+4:])

	if err := yaml.Unmarshal([]byte(fmContent), &fm); err != nil {
		return fm, body, err
	}

	// Parse tools from various formats
	fm.Tools = parseTools(fm.RawTools)

	return fm, body, nil
}

func parseTools(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		tools := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				tools = append(tools, strings.TrimSpace(s))
			}
		}
		return tools
	case string:
		parts := strings.Split(v, ",")
		tools := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				tools = append(tools, t)
			}
		}
		return tools
	}
	return nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/importer/ -run TestExtractFrontmatter -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/importer/frontmatter.go internal/importer/frontmatter_test.go
git commit -m "feat(import): add markdown frontmatter extraction"
```

---

### Task 6: Reverse Tool Mapping

**Files:**
- Create: `internal/importer/toolmap.go`
- Create: `internal/importer/toolmap_test.go`

**Step 1: Write the failing test**

```go
package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReverseMapTools_Claude(t *testing.T) {
	input := []string{"Read", "Write", "Bash", "Grep", "UnknownTool"}
	result := ReverseMapTools(input, FrameworkClaude)
	assert.Contains(t, result, "read_file")
	assert.Contains(t, result, "write_file")
	assert.Contains(t, result, "bash")
	assert.Contains(t, result, "grep")
	assert.NotContains(t, result, "UnknownTool") // unmapped tools dropped
}

func TestReverseMapTools_Copilot(t *testing.T) {
	input := []string{"read", "edit", "execute", "search"}
	result := ReverseMapTools(input, FrameworkCopilot)
	assert.Contains(t, result, "read_file")
	assert.Contains(t, result, "write_file")
	assert.Contains(t, result, "bash")
	assert.Contains(t, result, "grep")
}

func TestReverseMapTools_Unknown(t *testing.T) {
	input := []string{"Read", "Bash"} // Claude names work with Unknown too
	result := ReverseMapTools(input, FrameworkUnknown)
	assert.Contains(t, result, "read_file")
	assert.Contains(t, result, "bash")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -run TestReverseMapTools -v`
Expected: FAIL — undefined: ReverseMapTools

**Step 3: Write implementation**

```go
package importer

// reverseClaudeMap maps Claude Code tool names to Forgent generic tool names.
var reverseClaudeMap = map[string]string{
	"Read":      "read_file",
	"Write":     "write_file",
	"Edit":      "edit_file",
	"Grep":      "grep",
	"Glob":      "search",
	"Bash":      "bash",
	"WebFetch":  "web_fetch",
	"WebSearch": "web_search",
	"TodoWrite": "todo",
	"Task":      "task",
}

// reverseCopilotMap maps GitHub Copilot tool names to Forgent generic tool names.
var reverseCopilotMap = map[string]string{
	"read":    "read_file",
	"edit":    "write_file",
	"search":  "grep",
	"execute": "bash",
	"web":     "web_search",
}

// ReverseMapTools converts framework-specific tool names to Forgent generic names.
// Unknown tools are silently dropped.
func ReverseMapTools(tools []string, framework Framework) []string {
	maps := selectMaps(framework)
	seen := make(map[string]bool)
	var result []string

	for _, tool := range tools {
		for _, m := range maps {
			if generic, ok := m[tool]; ok && !seen[generic] {
				result = append(result, generic)
				seen[generic] = true
				break
			}
		}
	}
	return result
}

func selectMaps(framework Framework) []map[string]string {
	switch framework {
	case FrameworkClaude:
		return []map[string]string{reverseClaudeMap}
	case FrameworkCopilot:
		return []map[string]string{reverseCopilotMap}
	default:
		return []map[string]string{reverseClaudeMap, reverseCopilotMap}
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/importer/ -run TestReverseMapTools -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/importer/toolmap.go internal/importer/toolmap_test.go
git commit -m "feat(import): add reverse tool mapping (Claude + Copilot)"
```

---

### Task 7: LLM Prompt Construction

**Files:**
- Create: `internal/importer/prompt.go`
- Create: `internal/importer/prompt_test.go`

**Step 1: Write the failing test**

```go
package importer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildImportPrompt(t *testing.T) {
	source := Source{
		Name:    "review.md",
		Content: "# Review Agent\nReviews PRs",
	}
	fm := AgentFrontmatter{
		Name:        "review",
		Description: "Reviews PRs",
		Tools:       []string{"Read", "Grep"},
	}

	prompt := BuildImportPrompt(source, fm, "# Review Agent\nReviews PRs", nil)

	assert.Contains(t, prompt, "Forgent skill YAML schema")
	assert.Contains(t, prompt, "review")
	assert.Contains(t, prompt, "Reviews PRs")
	assert.Contains(t, prompt, `"skills"`)   // JSON output format
	assert.Contains(t, prompt, "consumes")
	assert.Contains(t, prompt, "produces")
}

func TestBuildRetryPrompt(t *testing.T) {
	feedback := []string{
		"skill 'review': missing guardrails.timeout",
		"skill 'review': score 45/100 (below 60 threshold)",
	}
	prompt := BuildRetryPrompt("original prompt", "original response", feedback)

	assert.Contains(t, prompt, "missing guardrails.timeout")
	assert.Contains(t, prompt, "score 45/100")
	assert.Contains(t, prompt, "Fix these issues")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -run TestBuild.*Prompt -v`
Expected: FAIL — undefined: BuildImportPrompt, BuildRetryPrompt

**Step 3: Write implementation**

```go
package importer

import (
	"fmt"
	"strings"
)

// BuildImportPrompt constructs the LLM prompt for agent-to-skill decomposition.
func BuildImportPrompt(source Source, fm AgentFrontmatter, body string, genericTools []string) string {
	var b strings.Builder

	b.WriteString("You are converting an agent definition into Forgent skill YAML specs.\n\n")

	// Schema reference
	b.WriteString("## Forgent skill YAML schema\n\n")
	b.WriteString("Each skill MUST have these fields:\n")
	b.WriteString("- skill: string (kebab-case name)\n")
	b.WriteString("- version: string (semver)\n")
	b.WriteString("- context: { consumes: [string], produces: [string], memory: short-term|conversation|long-term }\n")
	b.WriteString("- strategy: { tools: [string], approach: string, steps: [string] }\n")
	b.WriteString("- guardrails: array of rules (strings or maps with key/value)\n")
	b.WriteString("- observability: { trace_level: minimal|standard|detailed, metrics: [string] }\n")
	b.WriteString("- security: { filesystem: none|read-only|read-write|full, network: none|allowlist|full, secrets: [string] }\n")
	b.WriteString("- negotiation: { file_conflicts: yield|override|merge, priority: int }\n\n")

	// Tool names
	b.WriteString("## Available generic tool names\n")
	b.WriteString("read_file, write_file, edit_file, grep, search, bash, web_fetch, web_search, todo, task\n\n")

	// Agent composition schema
	b.WriteString("## Agent composition schema (optional, only if input has multiple responsibilities)\n")
	b.WriteString("- agent: string (kebab-case name)\n")
	b.WriteString("- skills: [string] (skill names)\n")
	b.WriteString("- orchestration: sequential|parallel|parallel-then-merge|adaptive\n")
	b.WriteString("- description: string\n")
	b.WriteString("- consumes: [string]\n")
	b.WriteString("- produces: [string]\n\n")

	// Input
	b.WriteString("## Input to analyze\n\n")
	if fm.Name != "" {
		b.WriteString(fmt.Sprintf("Source file: %s\n", source.Name))
		b.WriteString(fmt.Sprintf("Name: %s\n", fm.Name))
		b.WriteString(fmt.Sprintf("Description: %s\n", fm.Description))
		if len(genericTools) > 0 {
			b.WriteString(fmt.Sprintf("Tools (generic): %s\n", strings.Join(genericTools, ", ")))
		}
		b.WriteString("\n")
	}
	b.WriteString("### Full agent definition\n\n")
	b.WriteString(body)
	b.WriteString("\n\n")

	// Instructions
	b.WriteString("## Instructions\n\n")
	b.WriteString("1. Analyze this agent definition and identify distinct responsibilities.\n")
	b.WriteString("2. Create one Forgent skill YAML per responsibility.\n")
	b.WriteString("3. If the agent has multiple responsibilities, also create an agent composition.\n")
	b.WriteString("4. If the agent has a single responsibility, create only one skill (no agent).\n")
	b.WriteString("5. Set security to the minimum required permissions.\n")
	b.WriteString("6. Add meaningful guardrails (especially timeout for long-running tasks).\n\n")

	// Output format
	b.WriteString("## Output format\n\n")
	b.WriteString("Return ONLY valid JSON (no markdown fences, no explanation):\n")
	b.WriteString("```\n")
	b.WriteString(`{"skills": [{"yaml": "skill: name\nversion: ..."}], "agent": {"yaml": "agent: name\n..."} or null}`)
	b.WriteString("\n```\n")

	return b.String()
}

// BuildRetryPrompt appends validation feedback to the original prompt for a retry.
func BuildRetryPrompt(originalPrompt, originalResponse string, feedback []string) string {
	var b strings.Builder

	b.WriteString(originalPrompt)
	b.WriteString("\n\n## Previous attempt\n\n")
	b.WriteString("Your previous response:\n")
	b.WriteString(originalResponse)
	b.WriteString("\n\n## Validation feedback — Fix these issues\n\n")
	for _, f := range feedback {
		b.WriteString("- " + f + "\n")
	}
	b.WriteString("\nReturn the corrected JSON.\n")

	return b.String()
}
```

**Step 4: Run tests**

Run: `go test ./internal/importer/ -run TestBuild.*Prompt -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/importer/prompt.go internal/importer/prompt_test.go
git commit -m "feat(import): add LLM prompt construction with retry support"
```

---

### Task 8: Import Pipeline — Core Logic

**Files:**
- Create: `internal/importer/importer.go`
- Create: `internal/importer/importer_test.go`

**Step 1: Write the failing test**

Test with a mock LLM provider that returns known YAML:

```go
package importer

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testProvider struct {
	response string
}

func (p *testProvider) Complete(prompt string) (string, error) {
	return p.response, nil
}

func TestRunImport_SingleSkill(t *testing.T) {
	provider := &testProvider{
		response: `{"skills": [{"yaml": "skill: code-reviewer\nversion: \"0.1.0\"\ncontext:\n  consumes: [git_diff]\n  produces: [review_comments]\n  memory: short-term\nstrategy:\n  tools: [read_file, grep]\n  approach: sequential\n  steps:\n    - Read the diff\n    - Produce review comments\nguardrails:\n  - timeout: 120s\nobservability:\n  trace_level: minimal\n  metrics: [comments_count]\nsecurity:\n  filesystem: read-only\n  network: none\n  secrets: []\nnegotiation:\n  file_conflicts: yield\n  priority: 0"}], "agent": null}`,
	}

	result := RunImport(ImportOptions{
		Source:   ".claude/agents/review.md",
		Provider: provider,
		MinScore: 0, // No minimum for test
	})

	require.True(t, result.Success, "import should succeed: %s", result.Error)
	assert.Len(t, result.Skills, 1)
	assert.Equal(t, "code-reviewer", result.Skills[0].Skill.Skill)
	assert.Nil(t, result.Agent)
}

func TestRunImport_WithAgent(t *testing.T) {
	provider := &testProvider{
		response: `{"skills": [` +
			`{"yaml": "skill: linter\nversion: \"0.1.0\"\ncontext:\n  consumes: [source_code]\n  produces: [lint_results]\n  memory: short-term\nstrategy:\n  tools: [bash]\n  approach: sequential\nguardrails:\n  - timeout: 60s\nobservability:\n  trace_level: minimal\n  metrics: []\nsecurity:\n  filesystem: read-only\n  network: none\n  secrets: []\nnegotiation:\n  file_conflicts: yield\n  priority: 0"},` +
			`{"yaml": "skill: reviewer\nversion: \"0.1.0\"\ncontext:\n  consumes: [git_diff, lint_results]\n  produces: [review_comments]\n  memory: short-term\nstrategy:\n  tools: [read_file]\n  approach: sequential\nguardrails:\n  - timeout: 120s\nobservability:\n  trace_level: minimal\n  metrics: []\nsecurity:\n  filesystem: read-only\n  network: none\n  secrets: []\nnegotiation:\n  file_conflicts: yield\n  priority: 0"}` +
			`], "agent": {"yaml": "agent: ci-reviewer\nskills: [linter, reviewer]\norchestration: sequential\ndescription: CI review pipeline\nconsumes: [source_code, git_diff]\nproduces: [review_comments]"}}`,
	}

	result := RunImport(ImportOptions{
		Source:   ".claude/agents/ci-reviewer.md",
		Provider: provider,
		MinScore: 0,
	})

	require.True(t, result.Success, "import should succeed: %s", result.Error)
	assert.Len(t, result.Skills, 2)
	assert.NotNil(t, result.Agent)
	assert.Equal(t, "ci-reviewer", result.Agent.Agent.Agent)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -run TestRunImport -v`
Expected: FAIL — undefined: RunImport, ImportOptions

**Step 3: Write implementation**

```go
package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/analyzer"
	"github.com/mirandaguillaume/forgent/internal/linter"
	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"gopkg.in/yaml.v3"
)

// ImportOptions configures a single import run.
type ImportOptions struct {
	Source    string       // File path, directory path, or "vercel:name"
	Provider llm.Provider // LLM provider for decomposition
	MinScore int          // Minimum quality score (0 = no minimum)
	OutputDir string      // Output directory for generated files
}

// SkillResult holds a parsed skill and its validation results.
type SkillResult struct {
	Skill      model.SkillBehavior
	RawYAML    string
	Score      analyzer.SkillScore
	LintIssues []linter.LintResult
	LoopRisks  []analyzer.LoopRisk
}

// AgentResult holds a parsed agent and its validation results.
type AgentResult struct {
	Agent          model.AgentComposition
	RawYAML        string
	Score          analyzer.AgentScore
	DepIssues      []analyzer.DependencyIssue
	OrderingIssues []analyzer.OrderingIssue
}

// ImportResult is the output of RunImport.
type ImportResult struct {
	Success  bool
	Error    string
	Skills   []SkillResult
	Agent    *AgentResult
	Warnings []string
}

type llmResponse struct {
	Skills []struct {
		YAML string `json:"yaml"`
	} `json:"skills"`
	Agent *struct {
		YAML string `json:"yaml"`
	} `json:"agent"`
}

// RunImport executes the full import pipeline.
func RunImport(opts ImportOptions) ImportResult {
	// 1. Resolve sources
	sources, err := ResolveSources(opts.Source)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("resolve source: %v", err)}
	}

	// Process first source (multi-source batching is a future enhancement)
	source := sources[0]

	// 2. Extract frontmatter
	fm, body, err := ExtractFrontmatter(source.Content)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("parse frontmatter: %v", err)}
	}

	// Reverse map tools
	genericTools := ReverseMapTools(fm.Tools, source.Framework)

	// 3. LLM decomposition
	prompt := BuildImportPrompt(source, fm, body, genericTools)
	response, err := opts.Provider.Complete(prompt)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("LLM call failed: %v", err)}
	}

	// Parse LLM response
	skills, agent, err := parseLLMResponse(response)
	if err != nil {
		return ImportResult{Error: fmt.Sprintf("parse LLM response: %v", err)}
	}

	// 4. Validate
	result := validateImport(skills, agent)

	// Check if retry is needed
	if opts.MinScore > 0 {
		feedback := collectFeedback(result, opts.MinScore)
		if len(feedback) > 0 {
			retryPrompt := BuildRetryPrompt(prompt, response, feedback)
			retryResponse, err := opts.Provider.Complete(retryPrompt)
			if err == nil {
				retrySkills, retryAgent, err := parseLLMResponse(retryResponse)
				if err == nil {
					result = validateImport(retrySkills, retryAgent)
				}
			}
		}
	}

	result.Success = true
	return result
}

func parseLLMResponse(response string) ([]model.SkillBehavior, *model.AgentComposition, error) {
	// Strip markdown fences if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var resp llmResponse
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON from LLM: %w\nraw: %s", err, response)
	}

	var skills []model.SkillBehavior
	for i, s := range resp.Skills {
		var skill model.SkillBehavior
		if err := yaml.Unmarshal([]byte(s.YAML), &skill); err != nil {
			return nil, nil, fmt.Errorf("invalid YAML for skill %d: %w", i, err)
		}
		skills = append(skills, skill)
	}

	var agent *model.AgentComposition
	if resp.Agent != nil && resp.Agent.YAML != "" {
		var a model.AgentComposition
		if err := yaml.Unmarshal([]byte(resp.Agent.YAML), &a); err != nil {
			return nil, nil, fmt.Errorf("invalid YAML for agent: %w", err)
		}
		agent = &a
	}

	return skills, agent, nil
}

func validateImport(skills []model.SkillBehavior, agent *model.AgentComposition) ImportResult {
	var result ImportResult
	checker := &analyzer.DefaultGuardrailChecker{}

	for _, skill := range skills {
		sr := SkillResult{
			Skill:      skill,
			LintIssues: linter.LintSkill(skill),
			LoopRisks:  analyzer.DetectLoopRisks(skill, checker),
			Score:      analyzer.ScoreSkill(skill),
		}
		result.Skills = append(result.Skills, sr)

		// Collect validation warnings
		errs := model.ValidateSkill(skill)
		for _, e := range errs {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skill %q: %s", skill.Skill, e))
		}
	}

	if agent != nil {
		ar := AgentResult{
			Agent:     *agent,
			DepIssues: analyzer.CheckDependencies(*agent, skills),
		}

		// Build skill map for ordering check
		skillMap := make(map[string]model.SkillBehavior)
		for _, s := range skills {
			skillMap[s.Skill] = s
		}
		ar.OrderingIssues = analyzer.CheckSkillOrdering(*agent, skillMap)
		ar.Score = analyzer.ScoreAgent(*agent, skills)
		result.Agent = &ar

		errs := model.ValidateAgent(*agent)
		for _, e := range errs {
			result.Warnings = append(result.Warnings, fmt.Sprintf("agent %q: %s", agent.Agent, e))
		}
	}

	return result
}

func collectFeedback(result ImportResult, minScore int) []string {
	var feedback []string

	for _, sr := range result.Skills {
		for _, issue := range sr.LintIssues {
			if issue.Severity == linter.SeverityError {
				feedback = append(feedback, fmt.Sprintf("skill %q: lint error: %s", sr.Skill.Skill, issue.Message))
			}
		}
		if sr.Score.Total < minScore {
			feedback = append(feedback, fmt.Sprintf("skill %q: score %d/100 (below %d threshold)", sr.Skill.Skill, sr.Score.Total, minScore))
		}
	}

	if result.Agent != nil {
		for _, issue := range result.Agent.DepIssues {
			feedback = append(feedback, fmt.Sprintf("agent %q: %s", result.Agent.Agent.Agent, issue.Message))
		}
		if result.Agent.Score.Total < minScore {
			feedback = append(feedback, fmt.Sprintf("agent %q: score %d/100 (below %d threshold)", result.Agent.Agent.Agent, result.Agent.Score.Total, minScore))
		}
	}

	return feedback
}
```

**Step 4: Run tests**

Run: `go test ./internal/importer/ -v`
Expected: PASS

Note: The test uses a mock provider but the importer.go imports real packages (linter, analyzer). Make sure the source file for the test creates temp files so `ResolveSources` can read them. You may need to adjust the test to write temp `.md` files to disk before calling `RunImport`.

**Step 5: Commit**

```bash
git add internal/importer/importer.go internal/importer/importer_test.go
git commit -m "feat(import): add core import pipeline with validation"
```

---

### Task 9: File Writer

**Files:**
- Create: `internal/importer/writer.go`
- Create: `internal/importer/writer_test.go`

**Step 1: Write the failing test**

```go
package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteImportResult(t *testing.T) {
	dir := t.TempDir()
	result := ImportResult{
		Success: true,
		Skills: []SkillResult{
			{
				Skill: model.SkillBehavior{Skill: "code-reviewer", Version: "0.1.0"},
				RawYAML: "skill: code-reviewer\nversion: \"0.1.0\"\n",
			},
		},
	}

	written, err := WriteImportResult(result, dir)
	require.NoError(t, err)
	assert.Len(t, written, 1)

	content, err := os.ReadFile(filepath.Join(dir, "skills", "code-reviewer.skill.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "code-reviewer")
}

func TestWriteImportResult_WithAgent(t *testing.T) {
	dir := t.TempDir()
	result := ImportResult{
		Success: true,
		Skills: []SkillResult{
			{
				Skill:   model.SkillBehavior{Skill: "linter", Version: "0.1.0"},
				RawYAML: "skill: linter\nversion: \"0.1.0\"\n",
			},
		},
		Agent: &AgentResult{
			Agent:   model.AgentComposition{Agent: "ci-reviewer"},
			RawYAML: "agent: ci-reviewer\nskills: [linter]\n",
		},
	}

	written, err := WriteImportResult(result, dir)
	require.NoError(t, err)
	assert.Len(t, written, 2) // 1 skill + 1 agent

	_, err = os.Stat(filepath.Join(dir, "agents", "ci-reviewer.agent.yaml"))
	assert.NoError(t, err)
}

func TestWriteImportResult_ConflictDetection(t *testing.T) {
	dir := t.TempDir()
	// Pre-create a conflicting file
	skillDir := filepath.Join(dir, "skills")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "existing.skill.yaml"), []byte("old"), 0644)

	result := ImportResult{
		Success: true,
		Skills: []SkillResult{
			{
				Skill:   model.SkillBehavior{Skill: "existing", Version: "0.1.0"},
				RawYAML: "skill: existing\nversion: \"0.1.0\"\n",
			},
		},
	}

	_, err := WriteImportResult(result, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -run TestWriteImportResult -v`
Expected: FAIL — undefined: WriteImportResult

**Step 3: Write implementation**

```go
package importer

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteImportResult writes skills and agents to disk.
// Returns the list of written file paths.
func WriteImportResult(result ImportResult, outputDir string) ([]string, error) {
	var written []string

	skillsDir := filepath.Join(outputDir, "skills")
	agentsDir := filepath.Join(outputDir, "agents")

	// Write skills
	for _, sr := range result.Skills {
		path := filepath.Join(skillsDir, sr.Skill.Skill+".skill.yaml")
		if _, err := os.Stat(path); err == nil {
			return written, fmt.Errorf("file already exists: %s (use --force to overwrite)", path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return written, fmt.Errorf("create directory: %w", err)
		}
		yamlContent := sr.RawYAML
		if yamlContent == "" {
			yamlContent = skillToYAML(sr.Skill)
		}
		if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
			return written, fmt.Errorf("write %s: %w", path, err)
		}
		written = append(written, path)
	}

	// Write agent
	if result.Agent != nil {
		path := filepath.Join(agentsDir, result.Agent.Agent.Agent+".agent.yaml")
		if _, err := os.Stat(path); err == nil {
			return written, fmt.Errorf("file already exists: %s (use --force to overwrite)", path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return written, fmt.Errorf("create directory: %w", err)
		}
		yamlContent := result.Agent.RawYAML
		if yamlContent == "" {
			yamlContent = agentToYAML(result.Agent.Agent)
		}
		if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
			return written, fmt.Errorf("write %s: %w", path, err)
		}
		written = append(written, path)
	}

	return written, nil
}

func skillToYAML(skill model.SkillBehavior) string {
	data, _ := yaml.Marshal(skill)
	return string(data)
}

func agentToYAML(agent model.AgentComposition) string {
	data, _ := yaml.Marshal(agent)
	return string(data)
}
```

Note: Add the missing imports (`yaml`, `model`) in the actual file.

**Step 4: Run tests**

Run: `go test ./internal/importer/ -run TestWriteImportResult -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/importer/writer.go internal/importer/writer_test.go
git commit -m "feat(import): add file writer with conflict detection"
```

---

### Task 10: Preview / Dry-Run Output

**Files:**
- Create: `internal/importer/preview.go`
- Create: `internal/importer/preview_test.go`

**Step 1: Write the failing test**

```go
package importer

import (
	"bytes"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/analyzer"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestFormatPreview(t *testing.T) {
	result := ImportResult{
		Success: true,
		Skills: []SkillResult{
			{
				Skill: model.SkillBehavior{
					Skill:   "code-reviewer",
					Version: "0.1.0",
					Context: model.ContextFacet{
						Consumes: []string{"git_diff"},
						Produces: []string{"review_comments"},
					},
				},
				Score: analyzer.SkillScore{Skill: "code-reviewer", Total: 72},
			},
		},
	}

	var buf bytes.Buffer
	FormatPreview(result, &buf)
	output := buf.String()

	assert.Contains(t, output, "code-reviewer")
	assert.Contains(t, output, "72/100")
	assert.Contains(t, output, "git_diff")
	assert.Contains(t, output, "review_comments")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -run TestFormatPreview -v`
Expected: FAIL — undefined: FormatPreview

**Step 3: Write implementation**

```go
package importer

import (
	"fmt"
	"io"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/linter"
)

// FormatPreview writes a human-readable preview of the import result to w.
func FormatPreview(result ImportResult, w io.Writer) {
	skillCount := len(result.Skills)
	hasAgent := result.Agent != nil

	if hasAgent {
		fmt.Fprintf(w, "\n  Decomposition:\n")
		fmt.Fprintf(w, "    Input agent → %d skills + 1 agent\n\n", skillCount)
	} else {
		fmt.Fprintf(w, "\n  Result: %d skill(s)\n\n", skillCount)
	}

	fmt.Fprintf(w, "    Skills:\n")
	for _, sr := range result.Skills {
		fmt.Fprintf(w, "      %-45s score: %d/100\n", sr.Skill.Skill, sr.Score.Total)
		fmt.Fprintf(w, "        consumes: [%s]\n", strings.Join(sr.Skill.Context.Consumes, ", "))
		fmt.Fprintf(w, "        produces: [%s]\n", strings.Join(sr.Skill.Context.Produces, ", "))

		if sr.Skill.Security.Filesystem != "" || sr.Skill.Security.Network != "" {
			fmt.Fprintf(w, "        security: filesystem=%s, network=%s\n",
				sr.Skill.Security.Filesystem, sr.Skill.Security.Network)
		}

		// Lint issues
		hasErrors := false
		for _, issue := range sr.LintIssues {
			if issue.Severity == linter.SeverityError || issue.Severity == linter.SeverityWarning {
				fmt.Fprintf(w, "        ⚠ lint: %s\n", issue.Message)
				hasErrors = true
			}
		}
		// Loop risks
		for _, risk := range sr.LoopRisks {
			fmt.Fprintf(w, "        ⚠ loop: %s\n", risk.Message)
			hasErrors = true
		}
		if !hasErrors {
			fmt.Fprintf(w, "        ✓ all checks pass\n")
		}
		fmt.Fprintln(w)
	}

	if hasAgent {
		ar := result.Agent
		fmt.Fprintf(w, "    Agent:\n")
		fmt.Fprintf(w, "      %-45s score: %d/100\n", ar.Agent.Agent, ar.Score.Total)
		fmt.Fprintf(w, "        skills: [%s]\n", strings.Join(ar.Agent.Skills, ", "))
		fmt.Fprintf(w, "        orchestration: %s\n", ar.Agent.Orchestration)
		if len(ar.Agent.Consumes) > 0 {
			fmt.Fprintf(w, "        consumes: [%s]\n", strings.Join(ar.Agent.Consumes, ", "))
		}
		if len(ar.Agent.Produces) > 0 {
			fmt.Fprintf(w, "        produces: [%s]\n", strings.Join(ar.Agent.Produces, ", "))
		}

		hasIssues := false
		for _, issue := range ar.DepIssues {
			fmt.Fprintf(w, "        ⚠ dep: %s\n", issue.Message)
			hasIssues = true
		}
		for _, issue := range ar.OrderingIssues {
			fmt.Fprintf(w, "        ⚠ order: %s\n", issue.Message)
			hasIssues = true
		}
		if !hasIssues {
			fmt.Fprintf(w, "        ✓ dependencies satisfied\n")
			fmt.Fprintf(w, "        ✓ skill ordering valid\n")
		}
		fmt.Fprintln(w)
	}

	// Warnings
	for _, w2 := range result.Warnings {
		fmt.Fprintf(w, "  ⚠ %s\n", w2)
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/importer/ -run TestFormatPreview -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/importer/preview.go internal/importer/preview_test.go
git commit -m "feat(import): add dry-run preview output"
```

---

### Task 11: CLI Command — `forgent import`

**Files:**
- Create: `internal/cmd/import.go`
- Create: `internal/cmd/import_test.go`

**Step 1: Write the failing test**

```go
package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		provider string
		expected string
		wantErr  bool
	}{
		{
			name:     "FORGENT_API_KEY takes priority",
			env:      map[string]string{"FORGENT_API_KEY": "forgent-key"},
			provider: "anthropic",
			expected: "forgent-key",
		},
		{
			name:     "falls back to ANTHROPIC_API_KEY",
			env:      map[string]string{"ANTHROPIC_API_KEY": "anthropic-key"},
			provider: "anthropic",
			expected: "anthropic-key",
		},
		{
			name:     "falls back to OPENAI_API_KEY",
			env:      map[string]string{"OPENAI_API_KEY": "openai-key"},
			provider: "openai",
			expected: "openai-key",
		},
		{
			name:     "no key found",
			env:      map[string]string{},
			provider: "anthropic",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := resolveAPIKey(tt.provider, tt.env)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, key)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cmd/ -run TestResolveAPIKey -v`
Expected: FAIL — undefined: resolveAPIKey

**Step 3: Write implementation**

```go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/importer"
	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/spf13/cobra"
)

var (
	importProvider string
	importOutput   string
	importMinScore int
	importYes      bool
	importDryRun   bool
	importForce    bool
)

func init() {
	importCmd := &cobra.Command{
		Use:   "import <source>",
		Short: "Import agent definitions as Forgent skill specs",
		Long: `Convert existing agent markdown files (Claude Code, Copilot, etc.)
or Vercel registry skills into composable Forgent YAML specs.

Uses LLM-assisted decomposition to split monolithic agents into
focused skills, validated by Forgent's lint, doctor, and score tools.`,
		Args: cobra.ExactArgs(1),
		Run:  runImport,
	}

	importCmd.Flags().StringVarP(&importProvider, "provider", "p", "", "LLM provider (default: from FORGENT_LLM_PROVIDER or anthropic)")
	importCmd.Flags().StringVarP(&importOutput, "output", "o", ".", "Output directory")
	importCmd.Flags().IntVar(&importMinScore, "min-score", 60, "Minimum quality score (triggers LLM retry if below)")
	importCmd.Flags().BoolVar(&importYes, "yes", false, "Skip confirmation, write directly")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Show what would be generated (default behavior)")
	importCmd.Flags().BoolVar(&importForce, "force", false, "Write even if validation fails or files exist")

	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) {
	source := args[0]

	// Resolve provider
	providerName := importProvider
	if providerName == "" {
		providerName = os.Getenv("FORGENT_LLM_PROVIDER")
	}
	if providerName == "" {
		providerName = "anthropic"
	}

	// Resolve API key
	envMap := map[string]string{
		"FORGENT_API_KEY":   os.Getenv("FORGENT_API_KEY"),
		"ANTHROPIC_API_KEY": os.Getenv("ANTHROPIC_API_KEY"),
		"OPENAI_API_KEY":    os.Getenv("OPENAI_API_KEY"),
	}
	apiKey, err := resolveAPIKey(providerName, envMap)
	if err != nil {
		color.Red("Error: %v", err)
		color.Yellow("\nSet one of: FORGENT_API_KEY, ANTHROPIC_API_KEY, or OPENAI_API_KEY")
		os.Exit(1)
	}

	// Get provider
	provider, err := llm.GetProvider(providerName, apiKey)
	if err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}

	// Run import
	color.Cyan("  Analyzing: %s", source)
	color.Cyan("  Provider: %s\n", providerName)

	result := importer.RunImport(importer.ImportOptions{
		Source:    source,
		Provider:  provider,
		MinScore:  importMinScore,
		OutputDir: importOutput,
	})

	if !result.Success {
		color.Red("Error: %s", result.Error)
		os.Exit(1)
	}

	// Preview
	importer.FormatPreview(result, os.Stdout)

	// Dry-run is default unless --yes
	if !importYes {
		skillCount := len(result.Skills)
		agentCount := 0
		if result.Agent != nil {
			agentCount = 1
		}
		fmt.Printf("\n  Write %d skill(s) + %d agent(s)? [y/N] ", skillCount, agentCount)

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			color.Yellow("  Aborted.")
			return
		}
	}

	// Write
	written, err := importer.WriteImportResult(result, importOutput)
	if err != nil {
		if !importForce {
			color.Red("Error: %v", err)
			os.Exit(1)
		}
		color.Yellow("Warning: %v (--force: writing anyway)", err)
	}

	color.Green("\n  Written %d file(s):", len(written))
	for _, path := range written {
		fmt.Printf("    %s\n", path)
	}
}

// resolveAPIKey finds the API key from environment variables.
func resolveAPIKey(provider string, env map[string]string) (string, error) {
	// Priority 1: FORGENT_API_KEY
	if key := env["FORGENT_API_KEY"]; key != "" {
		return key, nil
	}

	// Priority 2: Provider-specific key
	providerEnvMap := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
	}
	if envVar, ok := providerEnvMap[provider]; ok {
		if key := env[envVar]; key != "" {
			return key, nil
		}
	}

	return "", fmt.Errorf("no API key found for provider %q", provider)
}
```

**Step 4: Run tests**

Run: `go test ./internal/cmd/ -run TestResolveAPIKey -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cmd/import.go internal/cmd/import_test.go
git commit -m "feat(import): add forgent import CLI command"
```

---

### Task 12: Integration Test — Golden File

**Files:**
- Create: `internal/importer/testdata/input/simple-reviewer.md`
- Create: `internal/importer/integration_test.go`

**Step 1: Create golden test input**

```markdown
---
name: code-reviewer
description: Reviews pull request code changes
tools: [Read, Grep, Bash]
---

# Code Reviewer

You are a code review agent. Your job is to:

1. Read the git diff
2. Check for common issues (security, performance, style)
3. Write review comments

## Guidelines

- Be constructive
- Focus on bugs over style
- Suggest specific fixes
```

**Step 2: Write integration test**

```go
package importer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_SimpleReviewer(t *testing.T) {
	// Read golden input
	input, err := os.ReadFile("testdata/input/simple-reviewer.md")
	require.NoError(t, err)

	// Mock provider returns a pre-built response
	provider := &testProvider{
		response: `{"skills": [{"yaml": "skill: code-reviewer\nversion: \"0.1.0\"\ncontext:\n  consumes: [git_diff, source_code]\n  produces: [review_comments]\n  memory: short-term\nstrategy:\n  tools: [read_file, grep, bash]\n  approach: sequential\n  steps:\n    - Read the git diff\n    - Check for security, performance, and style issues\n    - Write review comments with specific suggestions\nguardrails:\n  - timeout: 120s\n  - Be constructive and focus on bugs over style\nobservability:\n  trace_level: standard\n  metrics: [comments_count, issues_found]\nsecurity:\n  filesystem: read-only\n  network: none\n  secrets: []\nnegotiation:\n  file_conflicts: yield\n  priority: 0"}], "agent": null}`,
	}

	// Write input to temp file
	dir := t.TempDir()
	inputPath := dir + "/code-reviewer.md"
	os.WriteFile(inputPath, input, 0644)

	result := RunImport(ImportOptions{
		Source:    inputPath,
		Provider:  provider,
		MinScore:  0,
		OutputDir: dir,
	})

	require.True(t, result.Success, "import failed: %s", result.Error)
	assert.Len(t, result.Skills, 1)

	skill := result.Skills[0].Skill
	assert.Equal(t, "code-reviewer", skill.Skill)
	assert.Contains(t, skill.Context.Consumes, "git_diff")
	assert.Contains(t, skill.Context.Produces, "review_comments")
	assert.Contains(t, skill.Strategy.Tools, "read_file")
	assert.Equal(t, "read-only", string(skill.Security.Filesystem))
	assert.Nil(t, result.Agent)

	// Score should be reasonable
	assert.Greater(t, result.Skills[0].Score.Total, 40)

	// Write and verify output
	written, err := WriteImportResult(result, dir)
	require.NoError(t, err)
	assert.Len(t, written, 1)

	// Verify written file is valid YAML
	content, err := os.ReadFile(written[0])
	require.NoError(t, err)
	assert.Contains(t, string(content), "code-reviewer")
}
```

**Step 3: Run test**

Run: `go test ./internal/importer/ -run TestIntegration -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/importer/testdata/ internal/importer/integration_test.go
git commit -m "test(import): add integration test with golden file"
```

---

### Task 13: Run Full Test Suite + Fix

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All pass

**Step 2: Run vet**

Run: `go vet ./...`
Expected: No issues

**Step 3: Fix any compilation or test issues**

Iterate until all tests pass.

**Step 4: Commit fixes if any**

```bash
git add -A
git commit -m "fix(import): resolve test and compilation issues"
```

---

### Task 14: Manual Smoke Test

**Step 1: Build the binary**

Run: `go build ./cmd/forgent`

**Step 2: Test help output**

Run: `./forgent import --help`
Expected: Shows usage with all flags (--provider, --output, --min-score, --yes, --force, --dry-run)

**Step 3: Test with a real agent file (dry-run)**

If an agent `.md` file exists in the project:

Run: `./forgent import .claude/agents/ci-reviewer.md --dry-run` (requires API key)

Or test error handling:

Run: `./forgent import nonexistent.md`
Expected: Error message about file not found

**Step 4: Commit nothing** (smoke test only)

---

## Summary

| Task | Component | Files | Tests |
|------|-----------|-------|-------|
| 1 | LLM Provider Interface | `internal/llm/provider.go` | 1 test |
| 2 | Provider Registry | `internal/llm/registry.go` | 2 tests |
| 3 | Anthropic Provider | `internal/llm/anthropic.go` | 2 tests |
| 4 | Source Resolution | `internal/importer/source.go` | 4 tests |
| 5 | Frontmatter Extraction | `internal/importer/frontmatter.go` | 3 tests |
| 6 | Reverse Tool Mapping | `internal/importer/toolmap.go` | 3 tests |
| 7 | LLM Prompt Construction | `internal/importer/prompt.go` | 2 tests |
| 8 | Core Import Pipeline | `internal/importer/importer.go` | 2 tests |
| 9 | File Writer | `internal/importer/writer.go` | 3 tests |
| 10 | Preview Output | `internal/importer/preview.go` | 1 test |
| 11 | CLI Command | `internal/cmd/import.go` | 1 test |
| 12 | Integration Test | `internal/importer/integration_test.go` | 1 test |
| 13 | Full Suite | — | All |
| 14 | Smoke Test | — | Manual |
