package generator_test

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestToTitle(t *testing.T) {
	assert.Equal(t, "My Skill Name", generator.ToTitle("my-skill-name"))
	assert.Equal(t, "Simple", generator.ToTitle("simple"))
	assert.Equal(t, "", generator.ToTitle(""))
}

func TestCountWords(t *testing.T) {
	assert.Equal(t, 3, generator.CountWords("one two three"))
	assert.Equal(t, 0, generator.CountWords(""))
	assert.Equal(t, 0, generator.CountWords("   "))
}

func TestBuildSkillDescription(t *testing.T) {
	skill := model.SkillBehavior{
		Strategy: model.StrategyFacet{Approach: "analytical"},
		Context: model.ContextFacet{
			Consumes: []string{"source-code"},
			Produces: []string{"report"},
		},
	}
	desc := generator.BuildSkillDescription(skill)
	assert.Equal(t, "analytical-based skill consuming source-code to produce report", desc)
}

func TestBuildSkillDescription_NoConsumesProduces(t *testing.T) {
	skill := model.SkillBehavior{
		Strategy: model.StrategyFacet{Approach: "generative"},
	}
	desc := generator.BuildSkillDescription(skill)
	assert.Equal(t, "generative-based skill", desc)
}
