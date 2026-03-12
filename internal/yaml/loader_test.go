package yamlloader_test

import (
	"testing"

	yamlloader "github.com/mirandaguillaume/forgent/internal/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validSkillYAML = `
skill: test-skill
version: "1.0.0"
context:
  consumes: []
  produces: []
  memory: short-term
strategy:
  tools: []
  approach: sequential
guardrails: []

observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: none
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`

const validAgentYAML = `
agent: test-agent
skills:
  - skill-a
  - skill-b
orchestration: sequential
description: A test agent
`

func TestParseSkillYAML_Valid(t *testing.T) {
	skill, err := yamlloader.ParseSkillYAML(validSkillYAML)
	require.NoError(t, err)
	assert.Equal(t, "test-skill", skill.Skill)
	assert.Equal(t, "1.0.0", skill.Version)
	assert.Equal(t, "sequential", skill.Strategy.Approach)
}

func TestParseSkillYAML_InvalidSyntax(t *testing.T) {
	_, err := yamlloader.ParseSkillYAML("skill: [invalid yaml\n  broken:")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YAML")
}

func TestParseSkillYAML_ValidationError_EmptyName(t *testing.T) {
	yaml := `
skill: ""
version: "1.0.0"
context:
  consumes: []
  produces: []
  memory: short-term
strategy:
  tools: []
  approach: sequential
guardrails: []

observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: none
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`
	_, err := yamlloader.ParseSkillYAML(yaml)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
	assert.Contains(t, err.Error(), "skill name is required")
}

func TestParseAgentYAML_Valid(t *testing.T) {
	agent, err := yamlloader.ParseAgentYAML(validAgentYAML)
	require.NoError(t, err)
	assert.Equal(t, "test-agent", agent.Agent)
	assert.Equal(t, []string{"skill-a", "skill-b"}, agent.Skills)
	assert.Equal(t, "A test agent", agent.Description)
}

func TestParseAgentYAML_InvalidSyntax(t *testing.T) {
	_, err := yamlloader.ParseAgentYAML("agent: [broken yaml\n  bad:")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "YAML")
}

func TestParseAgentYAML_ValidationError_EmptySkills(t *testing.T) {
	yaml := `
agent: test-agent
skills: []
orchestration: sequential
`
	_, err := yamlloader.ParseAgentYAML(yaml)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
	assert.Contains(t, err.Error(), "at least one skill is required")
}
