package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/cmd"
	_ "github.com/mirandaguillaume/forgent/internal/generator/claude"
	_ "github.com/mirandaguillaume/forgent/internal/generator/copilot"
	"github.com/stretchr/testify/assert"
)

const testSkillYAML = `skill: test-skill
version: "1.0.0"
context:
  consumes: [input]
  produces: [output]
  memory: short-term
strategy:
  tools: [read_file, grep]
  approach: sequential
guardrails:
  - "timeout: 60s"
depends_on: []
observability:
  trace_level: standard
  metrics: [duration]
security:
  filesystem: read-only
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`

const testAgentYAML = `agent: test-agent
skills: [test-skill]
orchestration: sequential
description: "Test agent"
`

func TestRunBuild_Claude(t *testing.T) {
	skillsDir := t.TempDir()
	agentsDir := t.TempDir()
	outputDir := t.TempDir()

	os.WriteFile(filepath.Join(skillsDir, "test-skill.skill.yaml"), []byte(testSkillYAML), 0644)
	os.WriteFile(filepath.Join(agentsDir, "test-agent.agent.yaml"), []byte(testAgentYAML), 0644)

	result := cmd.RunBuild(skillsDir, agentsDir, outputDir, "claude")
	assert.True(t, result.Success)
	assert.Equal(t, 1, result.SkillsGenerated)
	assert.Equal(t, 1, result.AgentsGenerated)
	assert.FileExists(t, filepath.Join(outputDir, "skills", "test-skill", "SKILL.md"))
	assert.FileExists(t, filepath.Join(outputDir, "agents", "test-agent.md"))
}

func TestRunBuild_Copilot(t *testing.T) {
	skillsDir := t.TempDir()
	agentsDir := t.TempDir()
	outputDir := t.TempDir()

	os.WriteFile(filepath.Join(skillsDir, "test-skill.skill.yaml"), []byte(testSkillYAML), 0644)
	os.WriteFile(filepath.Join(agentsDir, "test-agent.agent.yaml"), []byte(testAgentYAML), 0644)

	result := cmd.RunBuild(skillsDir, agentsDir, outputDir, "copilot")
	assert.True(t, result.Success)
	assert.Equal(t, "copilot", result.Target)
	assert.FileExists(t, filepath.Join(outputDir, "skills", "test-skill", "SKILL.md"))
	assert.FileExists(t, filepath.Join(outputDir, "agents", "test-agent.agent.md"))
	assert.FileExists(t, filepath.Join(outputDir, "copilot-instructions.md"))
}

func TestRunBuild_UnknownTarget(t *testing.T) {
	result := cmd.RunBuild(".", ".", ".", "unknown")
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "unknown build target")
}

func TestRunBuild_EmptyDirs(t *testing.T) {
	result := cmd.RunBuild(t.TempDir(), t.TempDir(), t.TempDir(), "claude")
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.SkillsGenerated)
}

func TestGetOutputDir(t *testing.T) {
	assert.Equal(t, ".claude", cmd.GetOutputDir("claude", ""))
	assert.Equal(t, ".github", cmd.GetOutputDir("copilot", ""))
	assert.Equal(t, "custom", cmd.GetOutputDir("claude", "custom"))
}
