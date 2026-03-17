package forgent_test

import (
	"testing"

	_ "github.com/mirandaguillaume/forgent/internal/generator/forgent"
	"github.com/mirandaguillaume/forgent/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgentTarget_Registered(t *testing.T) {
	gen, err := spec.Get("forgent")
	require.NoError(t, err)
	assert.Equal(t, "forgent", gen.Target())
	assert.Equal(t, ".forgent", gen.DefaultOutputDir())
}

func TestForgentTarget_ImplementsAgentGenerator(t *testing.T) {
	gen, _ := spec.Get("forgent")
	_, ok := gen.(spec.AgentGenerator)
	assert.True(t, ok, "forgent generator must implement AgentGenerator")
}

func TestForgentTarget_NotSkillGenerator(t *testing.T) {
	gen, _ := spec.Get("forgent")
	_, ok := gen.(spec.SkillGenerator)
	assert.False(t, ok, "forgent generator should NOT implement SkillGenerator (skills are inlined)")
}

func TestForgentTarget_AgentPath(t *testing.T) {
	gen, _ := spec.Get("forgent")
	ag := gen.(spec.AgentGenerator)
	assert.Equal(t, "my_agent/main.go", ag.AgentPath("my-agent"))
	assert.Equal(t, "ci_reviewer/main.go", ag.AgentPath("ci-reviewer"))
}
