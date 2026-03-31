package static

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/stretchr/testify/assert"
)

func TestDuplicates_DetectsNearDuplicates(t *testing.T) {
	content := `## Rules
- Never expose API keys or secrets in the output
- Always validate inputs before processing
- Do not expose secrets or API keys in responses
- Always validate user inputs before processing them`

	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	check := &duplicatesCheck{}
	findings := check.Run(analysis)
	assert.True(t, len(findings) >= 1, "expected duplicate detection, got %d findings", len(findings))
}

func TestDuplicates_NoDuplicates(t *testing.T) {
	content := `## Rules
- Always run tests before committing
- Use gofmt for formatting
- Never skip the linter on pull requests`

	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	check := &duplicatesCheck{}
	findings := check.Run(analysis)
	assert.Empty(t, findings)
}

func TestDuplicates_SkipsCodeBlocks(t *testing.T) {
	content := "## Rules\n- Always run tests\n```\n- Always run tests\n```\n"
	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	check := &duplicatesCheck{}
	findings := check.Run(analysis)
	assert.Empty(t, findings, "duplicate inside code block should not match")
}

func TestDuplicates_NilAnalysis(t *testing.T) {
	check := &duplicatesCheck{}
	assert.Nil(t, check.Run(nil))
}

func TestJaccard(t *testing.T) {
	a := map[string]bool{"never": true, "expose": true, "api": true, "keys": true, "secrets": true}
	b := map[string]bool{"not": true, "expose": true, "secrets": true, "api": true, "keys": true}
	sim := jaccard(a, b)
	// intersection: expose, api, keys, secrets = 4
	// union: never, expose, api, keys, secrets, not = 6
	assert.InDelta(t, 4.0/6.0, sim, 0.01)
}
