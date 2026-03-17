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

// --- Mutation-testing-driven boundary tests ---

func TestLayers_SingleSkill(t *testing.T) {
	// Single source skill: should produce exactly 1 layer with 1 skill
	skills := []model.SkillBehavior{
		makeSkill("only", nil, []string{"out"}),
	}
	agent := makeAgent("single", []string{"only"}, nil, []string{"out"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	require.Equal(t, 1, len(layers), "single skill should produce 1 layer")
	assert.Equal(t, []string{"only"}, layers[0])
}

func TestWidth_SingleSkill(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("only", nil, []string{"out"}),
	}
	agent := makeAgent("single", []string{"only"}, nil, []string{"out"})

	g := BuildGraph(agent, skills)
	assert.Equal(t, 1, g.Width(), "single skill width should be 1")
}

func TestLayers_LongestPathWins(t *testing.T) {
	// Diamond where 'd' is reachable via short path (a→d) and long path (a→b→c→d).
	// Layer assignment should use the LONGEST path (layer 3), not shortest (layer 1).
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x", "p"}),
		makeSkill("b", []string{"x"}, []string{"y"}),
		makeSkill("c", []string{"y"}, []string{"z"}),
		makeSkill("d", []string{"z", "p"}, []string{"out"}), // consumes both z (long path) and p (short path from a)
	}
	agent := makeAgent("longpath", []string{"a", "b", "c", "d"}, nil, []string{"out"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	// a=layer0, b=layer1, c=layer2, d=layer3 (longest path from ⊤ through a→b→c→d)
	require.Equal(t, 4, len(layers), "should have 4 layers due to longest path")
	assert.Contains(t, layers[0], "a")
	assert.Contains(t, layers[1], "b")
	assert.Contains(t, layers[2], "c")
	assert.Contains(t, layers[3], "d")
}

func TestWidth_EmptyGraph(t *testing.T) {
	// No skills — graph has only ⊤ and ⊥
	agent := makeAgent("empty", nil, nil, nil)
	g := BuildGraph(agent, nil)
	assert.Equal(t, 0, g.Width(), "empty graph width should be 0")
	assert.Empty(t, g.Layers(), "empty graph should have no layers")
}

func TestLayers_ExcludesTopAndBottom(t *testing.T) {
	// ⊤ and ⊥ must never appear in any layer
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", []string{"x"}, []string{"y"}),
	}
	agent := makeAgent("pair", []string{"a", "b"}, nil, []string{"y"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	for i, layer := range layers {
		for _, name := range layer {
			assert.NotEqual(t, Top, name, "⊤ should not appear in layer %d", i)
			assert.NotEqual(t, Bottom, name, "⊥ should not appear in layer %d", i)
		}
	}
}

func TestWidth_ExactBoundary(t *testing.T) {
	// Width must be exactly the max layer size, not off-by-one.
	// Layer 0 has 2 skills, layer 1 has 1 skill → width = 2.
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", nil, []string{"y"}),
		makeSkill("c", []string{"x", "y"}, []string{"z"}),
	}
	agent := makeAgent("bounded", []string{"a", "b", "c"}, nil, []string{"z"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	// Verify actual layer sizes to confirm boundary
	require.Equal(t, 2, len(layers))
	assert.Equal(t, 2, len(layers[0]), "layer 0 should have exactly 2 skills")
	assert.Equal(t, 1, len(layers[1]), "layer 1 should have exactly 1 skill")
	assert.Equal(t, 2, g.Width(), "width should equal largest layer")
}

func TestLayers_NoTrailingEmpty(t *testing.T) {
	// Verify that layer output has no trailing empty slices.
	// Two parallel source skills producing disjoint outputs.
	skills := []model.SkillBehavior{
		makeSkill("x", nil, []string{"a"}),
		makeSkill("y", nil, []string{"b"}),
	}
	agent := makeAgent("flat", []string{"x", "y"}, nil, []string{"a", "b"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	require.Equal(t, 1, len(layers))
	for _, l := range layers {
		assert.NotEmpty(t, l, "no layer should be empty")
	}
}

func TestBuildGraph_DelegatesLayers(t *testing.T) {
	// After refactor, formal.Graph.Layers() must produce the same results as before.
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", []string{"x"}, []string{"y"}),
		makeSkill("c", []string{"y"}, []string{"z"}),
	}
	agent := makeAgent("chain", []string{"a", "b", "c"}, nil, []string{"z"})
	g := BuildGraph(agent, skills)
	layers := g.Layers()

	// 3 layers: a, b, c
	require.Equal(t, 3, len(layers))
	assert.Contains(t, layers[0], "a")
	assert.Contains(t, layers[1], "b")
	assert.Contains(t, layers[2], "c")
}

func TestBuildGraph_DAGField_NotNil(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", []string{"x"}, []string{"y"}),
	}
	agent := makeAgent("pair", []string{"a", "b"}, nil, []string{"y"})
	g := BuildGraph(agent, skills)

	assert.NotNil(t, g.DAG, "DAG field must be populated by BuildGraph")
	// DAG nodes must match skill nodes (no virtual ⊤/⊥ in pkg/dag)
	assert.Len(t, g.DAG.Nodes(), 2)
}
