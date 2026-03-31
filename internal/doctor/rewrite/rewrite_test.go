package rewrite

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements llm.Provider for testing.
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Complete(prompt string) (string, error) {
	return m.response, m.err
}

func TestRewrite_Success(t *testing.T) {
	original := []byte("---\nname: test\n---\n\n## Rules\n\n- Rule 1\n")
	findings := []llmutil.Finding{
		{Category: "guardrails", Issue: "No guardrails", SuggestedImprovement: "Add guardrails"},
	}

	expected := "---\nname: test\n---\n\n## Guardrails\n\n- Timeout: 30s\n\n## Rules\n\n- Rule 1\n"
	provider := &mockProvider{response: "```markdown\n" + expected + "\n```"}

	result, err := rewrite(original, findings, "claude", provider)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestRewrite_NoFences(t *testing.T) {
	original := []byte("# Agent")
	findings := []llmutil.Finding{
		{Category: "security", Issue: "No security", SuggestedImprovement: "Add security section"},
	}

	expected := "# Agent\n\n## Security\n\n- Read-only filesystem"
	provider := &mockProvider{response: expected}

	result, err := rewrite(original, findings, "claude", provider)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestRewrite_EmptyResponse(t *testing.T) {
	original := []byte("# Agent")
	findings := []llmutil.Finding{{Category: "guardrails", Issue: "test"}}
	provider := &mockProvider{response: "   \n  "}

	_, err := rewrite(original, findings, "claude", provider)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty rewrite")
}

func TestRewrite_NoFindings(t *testing.T) {
	original := []byte("# Agent")
	provider := &mockProvider{}

	result, err := rewrite(original, nil, "claude", provider)
	require.NoError(t, err)
	assert.Equal(t, original, result)
}

func TestRewrite_UTF8Preserved(t *testing.T) {
	original := []byte("# 你好世界 🎉\n\nContent with émojis and ñ")
	findings := []llmutil.Finding{{Category: "testing", Issue: "No tests"}}

	expected := "# 你好世界 🎉\n\nContent with émojis and ñ\n\n## Testing\n\n- Run tests"
	provider := &mockProvider{response: expected}

	result, err := rewrite(original, findings, "claude", provider)
	require.NoError(t, err)
	assert.Equal(t, expected, string(result))
}

func TestExtractFileContent_MarkdownFence(t *testing.T) {
	input := "Here's the rewritten file:\n\n```markdown\n---\nname: test\n---\nBody\n```\n\nDone."
	assert.Equal(t, "---\nname: test\n---\nBody", extractFileContent(input))
}

func TestExtractFileContent_NoFence(t *testing.T) {
	input := "---\nname: test\n---\nBody"
	assert.Equal(t, input, extractFileContent(input))
}

func TestBuildPrompt_TruncatesLargeContent(t *testing.T) {
	large := make([]byte, maxPromptContentBytes+1000)
	for i := range large {
		large[i] = 'x'
	}
	findings := []llmutil.Finding{{Category: "test", Issue: "test"}}

	prompt := buildPrompt(large, findings, "claude")
	assert.Contains(t, prompt, "[... truncated for prompt size ...]")
}

// rewrite is a test helper that calls the exported Rewrite function.
func rewrite(original []byte, findings []llmutil.Finding, format string, provider *mockProvider) ([]byte, error) {
	return Rewrite(original, findings, format, provider)
}
