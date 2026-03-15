package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsomorphism_CIReviewer(t *testing.T) {
	result, err := RunIsomorphism("../../skills", "../../agents")
	require.NoError(t, err)
	assert.True(t, result.SkillNamesMatch, "skill names should match across targets")
	assert.True(t, result.IOContractsMatch, "I/O contracts should match")
	assert.Equal(t, 1.0, result.StructureScore, "structure should be identical")
	// Should find all 6 skills
	assert.Len(t, result.ClaudeSkills, 6)
	assert.Len(t, result.CopilotSkills, 6)
	t.Logf("Claude skills: %v", result.ClaudeSkills)
	t.Logf("Copilot skills: %v", result.CopilotSkills)
}

func TestIsomorphism_SkillSignatures(t *testing.T) {
	// Verify specific I/O contracts are preserved across targets.
	result, err := RunIsomorphism("../../skills", "../../agents")
	require.NoError(t, err)

	// Find ts-linter in claude output
	var claudeLinter, copilotLinter *SkillSignature
	for i := range result.ClaudeSkills {
		if result.ClaudeSkills[i].Name == "ts-linter" {
			claudeLinter = &result.ClaudeSkills[i]
		}
	}
	for i := range result.CopilotSkills {
		if result.CopilotSkills[i].Name == "ts-linter" {
			copilotLinter = &result.CopilotSkills[i]
		}
	}
	require.NotNil(t, claudeLinter, "ts-linter should exist in claude output")
	require.NotNil(t, copilotLinter, "ts-linter should exist in copilot output")

	assert.Equal(t, claudeLinter.Consumes, copilotLinter.Consumes)
	assert.Equal(t, claudeLinter.Produces, copilotLinter.Produces)
}

func TestIsomorphism_EmptyProject(t *testing.T) {
	tmpSkills := t.TempDir()
	tmpAgents := t.TempDir()
	result, err := RunIsomorphism(tmpSkills, tmpAgents)
	require.NoError(t, err)
	assert.True(t, result.SkillNamesMatch, "empty → empty is a match")
	assert.Len(t, result.ClaudeSkills, 0)
	assert.Len(t, result.CopilotSkills, 0)
}
