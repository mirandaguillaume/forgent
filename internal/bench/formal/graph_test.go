package formal

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSkill(name string, consumes, produces []string) model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   name,
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: consumes,
			Produces: produces,
			Memory:   model.MemoryShortTerm,
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadOnly,
			Network:    model.NetworkNone,
		},
	}
}

func makeSkillWithSecurity(name string, consumes, produces []string, fs model.AccessLevel, net model.NetworkAccess) model.SkillBehavior {
	s := makeSkill(name, consumes, produces)
	s.Security.Filesystem = fs
	s.Security.Network = net
	return s
}

func makeAgent(name string, skills []string, consumes, produces []string) model.AgentComposition {
	return model.AgentComposition{
		Agent:         name,
		Skills:        skills,
		Orchestration: model.OrchestrationSequential,
		Consumes:      consumes,
		Produces:      produces,
	}
}

// --- Graph construction tests ---

func TestBuildGraph_CIReviewer(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("ts-linter", []string{"file_tree", "source_code"}, []string{"lint_results"}),
		makeSkill("type-checker", []string{"file_tree", "source_code"}, []string{"type_errors"}),
		makeSkill("tdd-runner", []string{"file_tree", "source_code"}, []string{"test_results"}),
		makeSkill("coverage-reporter", []string{"file_tree", "source_code"}, []string{"coverage_report"}),
		makeSkill("review-commenter", []string{"git_diff", "test_results", "lint_results"}, []string{"review_comments"}),
		makeSkill("risk-scorer", []string{"git_diff", "test_results", "lint_results"}, []string{"risk_score"}),
	}
	agent := makeAgent("ci-reviewer",
		[]string{"ts-linter", "type-checker", "tdd-runner", "coverage-reporter", "review-commenter", "risk-scorer"},
		nil,
		[]string{"lint_results", "type_errors", "test_results", "coverage_report", "review_comments", "risk_score"},
	)

	g := BuildGraph(agent, skills)

	// 6 skills + ⊤ + ⊥ = 8 nodes
	assert.Equal(t, 8, len(g.Nodes))

	// ⊤ should connect to source skills (those consuming external inputs)
	assert.Greater(t, len(g.Adj[Top]), 0, "⊤ should have outgoing edges")

	// All skills producing agent outputs should connect to ⊥
	assert.Greater(t, len(g.RevAdj[Bottom]), 0, "⊥ should have incoming edges")
}

func TestBuildGraph_SimpleChain(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", []string{"x"}, []string{"y"}),
		makeSkill("c", []string{"y"}, []string{"z"}),
	}
	agent := makeAgent("chain", []string{"a", "b", "c"}, nil, []string{"z"})

	g := BuildGraph(agent, skills)

	// a is a source skill → ⊤ → a
	assert.Contains(t, g.Adj[Top], "a")
	// a → b (b consumes x, a produces x)
	assert.Contains(t, g.Adj["a"], "b")
	// b → c
	assert.Contains(t, g.Adj["b"], "c")
	// c → ⊥ (c produces z which is agent output)
	assert.Contains(t, g.Adj["c"], Bottom)
}

func TestLayers_SimpleChain(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", []string{"x"}, []string{"y"}),
		makeSkill("c", []string{"y"}, []string{"z"}),
	}
	agent := makeAgent("chain", []string{"a", "b", "c"}, nil, []string{"z"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	require.Equal(t, 3, len(layers), "chain of 3 should have 3 layers")
	assert.Contains(t, layers[0], "a")
	assert.Contains(t, layers[1], "b")
	assert.Contains(t, layers[2], "c")
}

func TestLayers_ParallelSkills(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", nil, []string{"y"}),
		makeSkill("c", []string{"x", "y"}, []string{"z"}),
	}
	agent := makeAgent("diamond", []string{"a", "b", "c"}, nil, []string{"z"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	require.Equal(t, 2, len(layers))
	assert.Len(t, layers[0], 2, "layer 0 should have 2 parallel skills")
	assert.Contains(t, layers[0], "a")
	assert.Contains(t, layers[0], "b")
	assert.Contains(t, layers[1], "c")
}

func TestWidth_Diamond(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", nil, []string{"y"}),
		makeSkill("c", nil, []string{"w"}),
		makeSkill("d", []string{"x", "y", "w"}, []string{"z"}),
	}
	agent := makeAgent("wide", []string{"a", "b", "c", "d"}, nil, []string{"z"})

	g := BuildGraph(agent, skills)
	assert.Equal(t, 3, g.Width(), "width should be 3 (a, b, c are parallel)")
}
