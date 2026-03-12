package linter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/mirandaguillaume/forgent/pkg/model"
)

// minimalSkill returns a valid skill with tools and no guardrails.
func minimalSkill() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "test-skill",
		Version: "0.1.0",
		Context: model.ContextFacet{
			Produces: []string{"output"},
			Memory:   model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Tools:    []string{"Read"},
			Approach: "sequential",
		},
		Guardrails: nil,
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelMinimal,
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessNone,
			Network:    model.NetworkNone,
		},
		Negotiation: model.NegotiationFacet{
			FileConflicts: model.NegotiationYield,
		},
	}
}

// parseGuardrails parses a YAML list of guardrail rules for test use.
func parseGuardrails(t *testing.T, yamlContent string) []model.GuardrailRule {
	t.Helper()
	var rules []model.GuardrailRule
	err := yaml.Unmarshal([]byte(yamlContent), &rules)
	require.NoError(t, err)
	return rules
}

func TestNoEmptyTools_NoTools_Warning(t *testing.T) {
	skill := minimalSkill()
	skill.Strategy.Tools = nil

	result := noEmptyTools(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "no-empty-tools", result.Rule)
	assert.Equal(t, SeverityWarning, result.Severity)
	assert.Equal(t, "strategy", result.Facet)
	assert.Contains(t, result.Message, "test-skill")
}

func TestNoEmptyTools_WithTools_NoIssue(t *testing.T) {
	skill := minimalSkill()

	result := noEmptyTools(skill)

	assert.Nil(t, result)
}

func TestHasGuardrails_NoGuardrails_Warning(t *testing.T) {
	skill := minimalSkill()

	result := hasGuardrails(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "has-guardrails", result.Rule)
	assert.Equal(t, SeverityWarning, result.Severity)
	assert.Equal(t, "guardrails", result.Facet)
	assert.Contains(t, result.Message, "test-skill")
}

func TestObservableOutputs_ProducesNoMetrics_Warning(t *testing.T) {
	skill := minimalSkill()
	skill.Context.Produces = []string{"report.md"}
	skill.Observability.Metrics = nil

	result := observableOutputs(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "observable-outputs", result.Rule)
	assert.Equal(t, SeverityWarning, result.Severity)
	assert.Equal(t, "observability", result.Facet)
	assert.Contains(t, result.Message, "test-skill")
}

func TestObservableOutputs_NoProduces_NoIssue(t *testing.T) {
	skill := minimalSkill()
	skill.Context.Produces = nil
	skill.Observability.Metrics = nil

	result := observableOutputs(skill)

	assert.Nil(t, result)
}

func TestSecurityNeedsGuardrails_FullAccess_NoGuardrails_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Security.Filesystem = model.AccessFull
	skill.Guardrails = nil

	result := securityNeedsGuardrails(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "security-needs-guardrails", result.Rule)
	assert.Equal(t, SeverityError, result.Severity)
	assert.Equal(t, "security", result.Facet)
	assert.Contains(t, result.Message, "full")
}

func TestSecurityNeedsGuardrails_ReadWrite_NoGuardrails_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Security.Filesystem = model.AccessReadWrite
	skill.Guardrails = nil

	result := securityNeedsGuardrails(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "security-needs-guardrails", result.Rule)
	assert.Equal(t, SeverityError, result.Severity)
	assert.Contains(t, result.Message, "read-write")
}

func TestSecurityNeedsGuardrails_FullAccess_WithGuardrails_NoIssue(t *testing.T) {
	skill := minimalSkill()
	skill.Security.Filesystem = model.AccessFull
	skill.Guardrails = parseGuardrails(t, `- "timeout: 30s"`)

	result := securityNeedsGuardrails(skill)

	assert.Nil(t, result)
}

func TestHasWhenToUse_NoWhenToUse_Info(t *testing.T) {
	skill := minimalSkill()

	result := hasWhenToUse(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "has-when-to-use", result.Rule)
	assert.Equal(t, SeverityInfo, result.Severity)
	assert.Equal(t, "when_to_use", result.Facet)
}

func TestHasWhenToUse_WithTriggers_NoIssue(t *testing.T) {
	skill := minimalSkill()
	skill.WhenToUse = model.WhenToUseFacet{Triggers: []string{"bug"}}

	result := hasWhenToUse(skill)

	assert.Nil(t, result)
}

func TestSingleProducesOutput_ZeroProduces_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Context.Produces = nil

	result := singleProducesOutput(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "single-produces-output", result.Rule)
	assert.Equal(t, SeverityError, result.Severity)
	assert.Equal(t, "context", result.Facet)
	assert.Contains(t, result.Message, "test-skill")
	assert.Contains(t, result.Message, "0")
}

func TestSingleProducesOutput_MultipleProduces_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Context.Produces = []string{"output1", "output2"}

	result := singleProducesOutput(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "single-produces-output", result.Rule)
	assert.Equal(t, SeverityError, result.Severity)
	assert.Contains(t, result.Message, "test-skill")
	assert.Contains(t, result.Message, "2")
}

func TestSingleProducesOutput_OneProduces_NoIssue(t *testing.T) {
	skill := minimalSkill()

	result := singleProducesOutput(skill)

	assert.Nil(t, result)
}

func TestProducesMatchesDescription_ConjunctionAnd_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Strategy.Approach = "parse the file and generate output"

	result := producesMatchesDescription(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "produces-matches-description", result.Rule)
	assert.Equal(t, SeverityError, result.Severity)
	assert.Equal(t, "strategy", result.Facet)
	assert.Contains(t, result.Message, " and ")
}

func TestProducesMatchesDescription_ConjunctionEt_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Strategy.Approach = "analyser le fichier et produire le rapport"

	result := producesMatchesDescription(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "produces-matches-description", result.Rule)
	assert.Equal(t, SeverityError, result.Severity)
	assert.Contains(t, result.Message, " et ")
}

func TestProducesMatchesDescription_ConjunctionThen_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Strategy.Approach = "scan the code then report findings"

	result := producesMatchesDescription(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "produces-matches-description", result.Rule)
	assert.Contains(t, result.Message, " then ")
}

func TestProducesMatchesDescription_ConjunctionAmpersand_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Strategy.Approach = "lint & format the code"

	result := producesMatchesDescription(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "produces-matches-description", result.Rule)
	assert.Contains(t, result.Message, " & ")
}

func TestProducesMatchesDescription_CaseInsensitive_Error(t *testing.T) {
	skill := minimalSkill()
	skill.Strategy.Approach = "Parse the file AND generate output"

	result := producesMatchesDescription(skill)

	assert.NotNil(t, result)
	assert.Equal(t, "produces-matches-description", result.Rule)
}

func TestProducesMatchesDescription_NoConjunction_NoIssue(t *testing.T) {
	skill := minimalSkill()
	skill.Strategy.Approach = "sequential processing of inputs"

	result := producesMatchesDescription(skill)

	assert.Nil(t, result)
}

func TestLintSkill_CleanSkill_EmptyResults(t *testing.T) {
	skill := minimalSkill()
	skill.Guardrails = parseGuardrails(t, `- "timeout: 30s"`)
	skill.Observability.Metrics = []string{"latency"}
	skill.WhenToUse = model.WhenToUseFacet{Triggers: []string{"test"}}

	results := LintSkill(skill)

	assert.Empty(t, results)
}
