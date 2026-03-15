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
	t.Logf("Claude skills: %v", result.ClaudeSkills)
	t.Logf("Copilot skills: %v", result.CopilotSkills)
}
