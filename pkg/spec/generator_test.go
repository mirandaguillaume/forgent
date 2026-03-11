package spec_test

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGenerator struct{}

func (m *mockGenerator) Target() string            { return "mock" }
func (m *mockGenerator) DefaultOutputDir() string  { return ".mock" }
func (m *mockGenerator) GenerateSkill(_ model.SkillBehavior) string { return "skill-md" }
func (m *mockGenerator) GenerateAgent(_ model.AgentComposition, _ []model.SkillBehavior, _ string) string {
	return "agent-md"
}
func (m *mockGenerator) GenerateInstructions(_ []model.SkillBehavior, _ []model.AgentComposition) *string {
	return nil
}
func (m *mockGenerator) SkillPath(name string) string { return "skills/" + name + "/SKILL.md" }
func (m *mockGenerator) AgentPath(name string) string { return "agents/" + name + ".md" }
func (m *mockGenerator) InstructionsPath() *string    { return nil }

func TestRegisterAndGet(t *testing.T) {
	spec.Reset()
	spec.Register("mock", func() spec.TargetGenerator { return &mockGenerator{} })

	gen, err := spec.Get("mock")
	require.NoError(t, err)
	assert.Equal(t, "mock", gen.Target())
	assert.Equal(t, ".mock", gen.DefaultOutputDir())
	assert.Equal(t, "skill-md", gen.GenerateSkill(model.SkillBehavior{}))
}

func TestGet_Unknown(t *testing.T) {
	spec.Reset()
	_, err := spec.Get("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown build target")
}

func TestAvailable(t *testing.T) {
	spec.Reset()
	spec.Register("beta", func() spec.TargetGenerator { return &mockGenerator{} })
	spec.Register("alpha", func() spec.TargetGenerator { return &mockGenerator{} })

	targets := spec.Available()
	assert.Equal(t, []string{"alpha", "beta"}, targets) // sorted
}

func TestAvailable_Empty(t *testing.T) {
	spec.Reset()
	targets := spec.Available()
	assert.Empty(t, targets)
}

func TestReset(t *testing.T) {
	spec.Reset()
	spec.Register("test", func() spec.TargetGenerator { return &mockGenerator{} })
	assert.Len(t, spec.Available(), 1)
	spec.Reset()
	assert.Empty(t, spec.Available())
}

func TestGeneratorMethods(t *testing.T) {
	spec.Reset()
	spec.Register("mock", func() spec.TargetGenerator { return &mockGenerator{} })
	gen, _ := spec.Get("mock")

	assert.Equal(t, "skills/test/SKILL.md", gen.SkillPath("test"))
	assert.Equal(t, "agents/test.md", gen.AgentPath("test"))
	assert.Nil(t, gen.InstructionsPath())
	assert.Nil(t, gen.GenerateInstructions(nil, nil))
}
