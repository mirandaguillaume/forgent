package model_test

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSkillBehaviorYAMLParsing(t *testing.T) {
	input := `
skill: code-review
version: "1.0"
context:
  consumes:
    - pull-request
    - diff
  produces:
    - review-comment
  memory: conversation
strategy:
  tools:
    - github-api
    - linter
  approach: systematic
  steps:
    - read diff
    - check style
    - post comments
guardrails:
  - never approve without tests
  - max_comments: 10
depends_on:
  - skill: lint
    provides: lint-results
observability:
  trace_level: standard
  metrics:
    - review_time
    - comments_count
security:
  filesystem: read-only
  network: allowlist
  secrets:
    - GITHUB_TOKEN
  sandbox: container
negotiation:
  file_conflicts: merge
  priority: 5
`
	var sb model.SkillBehavior
	err := yaml.Unmarshal([]byte(input), &sb)
	require.NoError(t, err)

	assert.Equal(t, "code-review", sb.Skill)
	assert.Equal(t, "1.0", sb.Version)

	// Context
	assert.Equal(t, []string{"pull-request", "diff"}, sb.Context.Consumes)
	assert.Equal(t, []string{"review-comment"}, sb.Context.Produces)
	assert.Equal(t, model.MemoryConversation, sb.Context.Memory)

	// Strategy
	assert.Equal(t, []string{"github-api", "linter"}, sb.Strategy.Tools)
	assert.Equal(t, "systematic", sb.Strategy.Approach)
	assert.Equal(t, []string{"read diff", "check style", "post comments"}, sb.Strategy.Steps)

	// Guardrails
	require.Len(t, sb.Guardrails, 2)

	// Depends on
	require.Len(t, sb.DependsOn, 1)
	assert.Equal(t, "lint", sb.DependsOn[0].Skill)
	assert.Equal(t, "lint-results", sb.DependsOn[0].Provides)

	// Observability
	assert.Equal(t, model.TraceLevelStandard, sb.Observability.TraceLevel)
	assert.Equal(t, []string{"review_time", "comments_count"}, sb.Observability.Metrics)

	// Security
	assert.Equal(t, model.AccessReadOnly, sb.Security.Filesystem)
	assert.Equal(t, model.NetworkAllowlist, sb.Security.Network)
	assert.Equal(t, []string{"GITHUB_TOKEN"}, sb.Security.Secrets)
	assert.Equal(t, "container", sb.Security.Sandbox)

	// Negotiation
	assert.Equal(t, model.NegotiationMerge, sb.Negotiation.FileConflicts)
	assert.Equal(t, 5, sb.Negotiation.Priority)
}

func TestGuardrailRuleString(t *testing.T) {
	input := `- never approve without tests`
	var rules []model.GuardrailRule
	err := yaml.Unmarshal([]byte(input), &rules)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	val, ok := rules[0].StringValue()
	assert.True(t, ok)
	assert.Equal(t, "never approve without tests", val)

	_, ok = rules[0].MapValue()
	assert.False(t, ok)

	assert.True(t, rules[0].ContainsString("approve"))
	assert.False(t, rules[0].ContainsString("xyz"))
}

func TestGuardrailRuleMap(t *testing.T) {
	input := `- max_comments: 10`
	var rules []model.GuardrailRule
	err := yaml.Unmarshal([]byte(input), &rules)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	_, ok := rules[0].StringValue()
	assert.False(t, ok)

	m, ok := rules[0].MapValue()
	assert.True(t, ok)
	assert.Equal(t, 10, m["max_comments"])

	assert.True(t, rules[0].HasKey("max_comments"))
	assert.False(t, rules[0].HasKey("xyz"))
}

func TestMemoryTypeConstants(t *testing.T) {
	assert.Equal(t, model.MemoryType("short-term"), model.MemoryShortTerm)
	assert.Equal(t, model.MemoryType("conversation"), model.MemoryConversation)
	assert.Equal(t, model.MemoryType("long-term"), model.MemoryLongTerm)
}

func TestTraceLevelConstants(t *testing.T) {
	assert.Equal(t, model.TraceLevel("minimal"), model.TraceLevelMinimal)
	assert.Equal(t, model.TraceLevel("standard"), model.TraceLevelStandard)
	assert.Equal(t, model.TraceLevel("detailed"), model.TraceLevelDetailed)
}

func TestAccessLevelConstants(t *testing.T) {
	assert.Equal(t, model.AccessLevel("none"), model.AccessNone)
	assert.Equal(t, model.AccessLevel("read-only"), model.AccessReadOnly)
	assert.Equal(t, model.AccessLevel("read-write"), model.AccessReadWrite)
	assert.Equal(t, model.AccessLevel("full"), model.AccessFull)
}

func TestNetworkAccessConstants(t *testing.T) {
	assert.Equal(t, model.NetworkAccess("none"), model.NetworkNone)
	assert.Equal(t, model.NetworkAccess("allowlist"), model.NetworkAllowlist)
	assert.Equal(t, model.NetworkAccess("full"), model.NetworkFull)
}

func TestNegotiationStrategyConstants(t *testing.T) {
	assert.Equal(t, model.NegotiationStrategy("yield"), model.NegotiationYield)
	assert.Equal(t, model.NegotiationStrategy("override"), model.NegotiationOverride)
	assert.Equal(t, model.NegotiationStrategy("merge"), model.NegotiationMerge)
}
