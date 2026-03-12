package analyzer

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
)

func makeSkillWithDeps(name string, deps []model.Dependency, produces []string) model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   name,
		Version: "1.0.0",
		Context: model.ContextFacet{
			Produces: produces,
			Memory:   model.MemoryShortTerm,
		},
		DependsOn: deps,
	}
}

func TestCheckDependencies_NoIssues(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", nil, []string{"data"}),
		makeSkillWithDeps("skill-b", nil, []string{"result"}),
	}

	issues := CheckDependencies(skills)
	assert.Empty(t, issues)
}

func TestCheckDependencies_MissingDependency(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", []model.Dependency{
			{Skill: "nonexistent", Provides: "data"},
		}, nil),
	}

	issues := CheckDependencies(skills)
	assert.Len(t, issues, 1)
	assert.Equal(t, IssueMissing, issues[0].Type)
	assert.Equal(t, "skill-a", issues[0].Skill)
	assert.Contains(t, issues[0].Message, "nonexistent")
}

func TestCheckDependencies_CircularDependency(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", []model.Dependency{
			{Skill: "skill-b", Provides: "data"},
		}, []string{"result"}),
		makeSkillWithDeps("skill-b", []model.Dependency{
			{Skill: "skill-a", Provides: "result"},
		}, []string{"data"}),
	}

	issues := CheckDependencies(skills)

	hasCircular := false
	for _, issue := range issues {
		if issue.Type == IssueCircular {
			hasCircular = true
			assert.Contains(t, issue.Message, "Circular dependency detected")
		}
	}
	assert.True(t, hasCircular, "expected at least one circular dependency issue")
}

func TestCheckDependencies_UnmetContext(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", []model.Dependency{
			{Skill: "skill-b", Provides: "missing-output"},
		}, nil),
		makeSkillWithDeps("skill-b", nil, []string{"actual-output"}),
	}

	issues := CheckDependencies(skills)

	hasUnmet := false
	for _, issue := range issues {
		if issue.Type == IssueUnmetContext {
			hasUnmet = true
			assert.Equal(t, "skill-a", issue.Skill)
			assert.Contains(t, issue.Message, "missing-output")
			assert.Contains(t, issue.Message, "skill-b")
		}
	}
	assert.True(t, hasUnmet, "expected at least one unmet context issue")
}

func TestCheckDependencies_ValidContext(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", []model.Dependency{
			{Skill: "skill-b", Provides: "data"},
		}, nil),
		makeSkillWithDeps("skill-b", nil, []string{"data"}),
	}

	issues := CheckDependencies(skills)

	for _, issue := range issues {
		assert.NotEqual(t, IssueUnmetContext, issue.Type)
	}
}

// --- Tests for individual SRP-extracted functions ---

func TestCheckMissingDependencies_Missing(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", []model.Dependency{
			{Skill: "nonexistent", Provides: "data"},
		}, nil),
	}

	issues := CheckMissingDependencies(skills)
	assert.Len(t, issues, 1)
	assert.Equal(t, IssueMissing, issues[0].Type)
	assert.Equal(t, "skill-a", issues[0].Skill)
	assert.Contains(t, issues[0].Message, "nonexistent")
}

func TestCheckMissingDependencies_NoIssues(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", nil, []string{"data"}),
		makeSkillWithDeps("skill-b", []model.Dependency{
			{Skill: "skill-a", Provides: "data"},
		}, nil),
	}

	issues := CheckMissingDependencies(skills)
	assert.Empty(t, issues)
}

func TestCheckCircularDependencies_Cycle(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", []model.Dependency{
			{Skill: "skill-b", Provides: "data"},
		}, []string{"result"}),
		makeSkillWithDeps("skill-b", []model.Dependency{
			{Skill: "skill-a", Provides: "result"},
		}, []string{"data"}),
	}

	issues := CheckCircularDependencies(skills)
	assert.NotEmpty(t, issues)
	assert.Equal(t, IssueCircular, issues[0].Type)
	assert.Contains(t, issues[0].Message, "Circular dependency detected")
}

func TestCheckCircularDependencies_NoCycle(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", nil, []string{"data"}),
		makeSkillWithDeps("skill-b", []model.Dependency{
			{Skill: "skill-a", Provides: "data"},
		}, nil),
	}

	issues := CheckCircularDependencies(skills)
	assert.Empty(t, issues)
}

func TestCheckUnmetContext_Unmet(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", []model.Dependency{
			{Skill: "skill-b", Provides: "missing-output"},
		}, nil),
		makeSkillWithDeps("skill-b", nil, []string{"actual-output"}),
	}

	issues := CheckUnmetContext(skills)
	assert.Len(t, issues, 1)
	assert.Equal(t, IssueUnmetContext, issues[0].Type)
	assert.Equal(t, "skill-a", issues[0].Skill)
	assert.Contains(t, issues[0].Message, "missing-output")
	assert.Contains(t, issues[0].Message, "skill-b")
}

func TestCheckUnmetContext_Valid(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkillWithDeps("skill-a", []model.Dependency{
			{Skill: "skill-b", Provides: "data"},
		}, nil),
		makeSkillWithDeps("skill-b", nil, []string{"data"}),
	}

	issues := CheckUnmetContext(skills)
	assert.Empty(t, issues)
}
