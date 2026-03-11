package analyzer

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func makeLoopSkill(name string, consumes, produces []string, memory model.MemoryType, guardrails []model.GuardrailRule) model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   name,
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: consumes,
			Produces: produces,
			Memory:   memory,
		},
		Guardrails: guardrails,
	}
}

func guardrailFromString(t *testing.T, s string) model.GuardrailRule {
	t.Helper()
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"}
	var gr model.GuardrailRule
	err := gr.UnmarshalYAML(node)
	assert.NoError(t, err)
	return gr
}

func guardrailFromMap(t *testing.T, key, value string) model.GuardrailRule {
	t.Helper()
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: value, Tag: "!!str"},
		},
	}
	var gr model.GuardrailRule
	err := gr.UnmarshalYAML(node)
	assert.NoError(t, err)
	return gr
}

func TestDetectLoopRisks_SelfReference(t *testing.T) {
	skill := makeLoopSkill("self-ref", []string{"data", "extra"}, []string{"data"}, model.MemoryShortTerm, nil)

	risks := DetectLoopRisks(skill)

	hasSelfRef := false
	for _, r := range risks {
		if r.Type == LoopSelfReference {
			hasSelfRef = true
			assert.Equal(t, "error", r.Severity)
			assert.Contains(t, r.Message, "data")
			assert.Contains(t, r.Message, "infinite loops")
		}
	}
	assert.True(t, hasSelfRef, "expected self-reference risk")
}

func TestDetectLoopRisks_NoTimeout(t *testing.T) {
	skill := makeLoopSkill("no-timeout", []string{"input"}, []string{"output"}, model.MemoryConversation, nil)

	risks := DetectLoopRisks(skill)

	hasNoTimeout := false
	for _, r := range risks {
		if r.Type == LoopNoTimeout {
			hasNoTimeout = true
			assert.Equal(t, "warning", r.Severity)
			assert.Contains(t, r.Message, "conversation")
			assert.Contains(t, r.Message, "timeout")
		}
	}
	assert.True(t, hasNoTimeout, "expected no-timeout risk")
}

func TestDetectLoopRisks_CleanSkill(t *testing.T) {
	skill := makeLoopSkill("clean", []string{"input"}, []string{"output"}, model.MemoryShortTerm, nil)

	risks := DetectLoopRisks(skill)
	assert.Empty(t, risks)
}

func TestDetectLoopRisks_TimeoutMapGuardrail(t *testing.T) {
	gr := guardrailFromMap(t, "timeout", "5min")
	skill := makeLoopSkill("with-timeout", []string{"input"}, []string{"output"}, model.MemoryConversation, []model.GuardrailRule{gr})

	risks := DetectLoopRisks(skill)

	for _, r := range risks {
		assert.NotEqual(t, LoopNoTimeout, r.Type, "should not flag no-timeout when timeout guardrail exists")
	}
}

func TestDetectLoopRisks_TimeoutStringGuardrail(t *testing.T) {
	gr := guardrailFromString(t, "timeout: 5 minutes")
	skill := makeLoopSkill("with-timeout-str", []string{"input"}, []string{"output"}, model.MemoryConversation, []model.GuardrailRule{gr})

	risks := DetectLoopRisks(skill)

	for _, r := range risks {
		assert.NotEqual(t, LoopNoTimeout, r.Type, "should not flag no-timeout when timeout string guardrail exists")
	}
}

func TestDetectLoopRisks_LongTermMemoryNoTimeout(t *testing.T) {
	skill := makeLoopSkill("long-term", []string{"input"}, []string{"output"}, model.MemoryLongTerm, nil)

	risks := DetectLoopRisks(skill)

	hasNoTimeout := false
	for _, r := range risks {
		if r.Type == LoopNoTimeout {
			hasNoTimeout = true
			assert.Contains(t, r.Message, "long-term")
		}
	}
	assert.True(t, hasNoTimeout, "expected no-timeout risk for long-term memory")
}
