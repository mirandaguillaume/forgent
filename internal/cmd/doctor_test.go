package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const doctorValidSkillYAML = `skill: healthy-skill
version: "0.1.0"
context:
  consumes: []
  produces: []
  memory: short-term
strategy:
  tools:
    - Read
  approach: sequential
guardrails:
  - "timeout: 30s"
depends_on: []
observability:
  trace_level: minimal
  metrics:
    - latency
security:
  filesystem: none
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`

const doctorInvalidSkillYAML = `skill: ""
version: ""
`

const doctorLintIssuesSkillYAML = `skill: risky-skill
version: "0.1.0"
context:
  consumes: []
  produces:
    - output
  memory: short-term
strategy:
  tools: []
  approach: sequential
guardrails: []
depends_on: []
observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: read-write
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`

func TestRunDoctor_ValidSkills_HighScore(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "healthy.skill.yaml"), []byte(doctorValidSkillYAML), 0644)
	require.NoError(t, err)

	report := RunDoctor(dir, "")

	assert.Len(t, report.Skills, 1)
	assert.Empty(t, report.ParseErrors)
	assert.Empty(t, report.DependencyIssues)
	assert.Equal(t, 100, report.Score)
}

func TestRunDoctor_InvalidSkill_ParseErrors(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "bad.skill.yaml"), []byte(doctorInvalidSkillYAML), 0644)
	require.NoError(t, err)

	report := RunDoctor(dir, "")

	assert.Empty(t, report.Skills)
	assert.Len(t, report.ParseErrors, 1)
	assert.Equal(t, "bad.skill.yaml", report.ParseErrors[0].File)
	assert.Less(t, report.Score, 100)
}

func TestRunDoctor_EmptyDirectory_ZeroSkills(t *testing.T) {
	dir := t.TempDir()

	report := RunDoctor(dir, "")

	assert.Empty(t, report.Skills)
	assert.Empty(t, report.ParseErrors)
	assert.Equal(t, 100, report.Score) // No issues = perfect score
}

func TestRunDoctor_LintIssues_Populated(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "risky.skill.yaml"), []byte(doctorLintIssuesSkillYAML), 0644)
	require.NoError(t, err)

	report := RunDoctor(dir, "")

	assert.Len(t, report.Skills, 1)
	assert.Empty(t, report.ParseErrors)

	issues, ok := report.LintIssues["risky-skill"]
	assert.True(t, ok, "expected lint issues for risky-skill")
	assert.Greater(t, len(issues), 0)
}

func TestRunDoctor_NonExistentDirectory(t *testing.T) {
	report := RunDoctor("/nonexistent/path", "")

	assert.Empty(t, report.Skills)
	assert.Empty(t, report.ParseErrors)
	assert.Equal(t, 100, report.Score)
}

func TestRunDoctor_WithAgents_OrderingIssues(t *testing.T) {
	skillsDir := t.TempDir()
	agentsDir := t.TempDir()

	// Create two skills with data flow
	skill1 := `skill: producer
version: "0.1.0"
context:
  consumes: []
  produces:
    - data
  memory: short-term
strategy:
  tools:
    - Read
  approach: sequential
guardrails:
  - "timeout: 30s"
depends_on: []
observability:
  trace_level: minimal
  metrics:
    - latency
security:
  filesystem: none
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`
	skill2 := `skill: consumer
version: "0.1.0"
context:
  consumes:
    - data
  produces: []
  memory: short-term
strategy:
  tools:
    - Read
  approach: sequential
guardrails:
  - "timeout: 30s"
depends_on: []
observability:
  trace_level: minimal
  metrics:
    - latency
security:
  filesystem: none
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`
	// Agent with wrong ordering: consumer before producer
	agentYAML := `agent: bad-pipeline
skills:
  - consumer
  - producer
orchestration: sequential
description: "A pipeline that runs consumer then producer"
`

	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "producer.skill.yaml"), []byte(skill1), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "consumer.skill.yaml"), []byte(skill2), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "bad-pipeline.agent.yaml"), []byte(agentYAML), 0644))

	report := RunDoctor(skillsDir, agentsDir)

	assert.Len(t, report.Skills, 2)
	assert.Greater(t, len(report.OrderingIssues), 0)
}
