package formal

import (
	"sort"
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- P10: Reachability ---

func TestP10_Reachability_SimpleChain(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", []string{"x"}, []string{"y"}),
		makeSkill("c", []string{"y"}, []string{"z"}),
	}
	agent := makeAgent("test", []string{"a", "b", "c"}, nil, []string{"z"})
	g := BuildGraph(agent, skills)

	for _, s := range skills {
		assert.True(t, g.Reachable(Top, s.Skill),
			"skill %q should be reachable from ⊤", s.Skill)
	}
	// ⊥ reachable from every skill that produces agent output
	assert.True(t, g.Reachable("c", Bottom), "⊥ should be reachable from c")
	// Transitive: ⊥ reachable from a through the chain
	assert.True(t, g.Reachable("a", Bottom), "⊥ should be transitively reachable from a")
}

func TestP10_Reachability_SourceSkill(t *testing.T) {
	// Skill with C=∅ (no consumes) must still be reachable from ⊤.
	skills := []model.SkillBehavior{
		makeSkill("source", nil, []string{"data"}),
		makeSkill("sink", []string{"data"}, []string{"result"}),
	}
	agent := makeAgent("test", []string{"source", "sink"}, nil, []string{"result"})
	g := BuildGraph(agent, skills)

	assert.True(t, g.Reachable(Top, "source"))
	assert.True(t, g.Reachable(Top, "sink"))
}

func TestP10_Reachability_CIReviewer(t *testing.T) {
	// Full ci-reviewer graph: all 6 skills reachable from ⊤,
	// ⊥ reachable from all output-producing skills.
	skills, agent := ciReviewerFixture()
	g := BuildGraph(agent, skills)

	for _, s := range skills {
		assert.True(t, g.Reachable(Top, s.Skill),
			"skill %q should be reachable from ⊤", s.Skill)
	}
	// All skills produce agent outputs in ci-reviewer
	for _, s := range skills {
		assert.True(t, g.Reachable(s.Skill, Bottom),
			"⊥ should be reachable from %q", s.Skill)
	}
}

func TestP10_Reachability_NotReachable(t *testing.T) {
	// Skill b does NOT reach a (no backward reachability in a DAG).
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", []string{"x"}, []string{"y"}),
	}
	agent := makeAgent("test", []string{"a", "b"}, nil, []string{"y"})
	g := BuildGraph(agent, skills)

	assert.False(t, g.Reachable("b", "a"), "b should NOT reach a in a DAG")
}

// --- Corollary 3.1: Layer Decomposition ---

func TestCor31_LayerDecomposition_CIReviewer(t *testing.T) {
	skills, agent := ciReviewerFixture()
	g := BuildGraph(agent, skills)
	layers := g.Layers()

	// 1. Verify partition: every skill appears exactly once
	seen := make(map[string]bool)
	for _, layer := range layers {
		for _, s := range layer {
			assert.False(t, seen[s], "skill %q appears in multiple layers", s)
			seen[s] = true
		}
	}
	for _, s := range skills {
		assert.True(t, seen[s.Skill], "skill %q missing from layers", s.Skill)
	}

	// 2. Verify intra-layer independence: no edge between skills in same layer
	for li, layer := range layers {
		layerSet := toSet(layer)
		for _, s := range layer {
			for _, neighbor := range g.Adj[s] {
				if neighbor == Bottom {
					continue
				}
				assert.False(t, layerSet[neighbor],
					"intra-layer edge %s → %s in layer %d", s, neighbor, li)
			}
		}
	}

	// 3. Verify expected structure: layer 0 has 4 independent skills,
	//    layer 1 has 2 dependent skills
	require.Equal(t, 2, len(layers), "ci-reviewer should have 2 layers")
	sort.Strings(layers[0])
	sort.Strings(layers[1])
	assert.Equal(t, []string{"coverage-reporter", "tdd-runner", "ts-linter", "type-checker"}, layers[0])
	assert.Equal(t, []string{"review-commenter", "risk-scorer"}, layers[1])
}

// --- Corollary 3.2: Dilworth Width ---

func TestCor32_DilworthWidth(t *testing.T) {
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", nil, []string{"y"}),
		makeSkill("c", []string{"x", "y"}, []string{"z"}),
	}
	agent := makeAgent("test", []string{"a", "b", "c"}, nil, []string{"z"})

	g := BuildGraph(agent, skills)
	assert.Equal(t, 2, g.Width(), "a and b are parallel → width 2")
}

func TestCor32_DilworthWidth_CIReviewer(t *testing.T) {
	skills, agent := ciReviewerFixture()
	g := BuildGraph(agent, skills)
	// 4 skills in layer 0 → width 4
	assert.Equal(t, 4, g.Width())
}

// --- P11: Skill Fusion ---

func TestP11_Fusion_ExclusivelyConnected(t *testing.T) {
	// F1+F2+F3 all satisfied → fusible
	s1 := makeSkill("producer", nil, []string{"intermediate"})
	s2 := makeSkill("consumer", []string{"intermediate"}, []string{"result"})
	all := []model.SkillBehavior{s1, s2}
	agent := makeAgent("test", []string{"producer", "consumer"}, nil, []string{"result"})

	assert.True(t, CanFuse(s1, s2, agent, all))
}

func TestP11_Fusion_SharedConsumer_F2Violated(t *testing.T) {
	// Two consumers of intermediate → F2 violated
	s1 := makeSkill("producer", nil, []string{"intermediate"})
	s2 := makeSkill("consumer1", []string{"intermediate"}, []string{"r1"})
	s3 := makeSkill("consumer2", []string{"intermediate"}, []string{"r2"})
	all := []model.SkillBehavior{s1, s2, s3}
	agent := makeAgent("test", []string{"producer", "consumer1", "consumer2"}, nil, []string{"r1", "r2"})

	assert.False(t, CanFuse(s1, s2, agent, all))
}

func TestP11_Fusion_AgentOutput_F3Violated(t *testing.T) {
	// intermediate is an agent output → F3 violated
	s1 := makeSkill("producer", nil, []string{"intermediate"})
	s2 := makeSkill("consumer", []string{"intermediate"}, []string{"result"})
	all := []model.SkillBehavior{s1, s2}
	agent := makeAgent("test", []string{"producer", "consumer"}, nil, []string{"intermediate", "result"})

	assert.False(t, CanFuse(s1, s2, agent, all))
}

func TestP11_Fusion_NotConsumed_F1Violated(t *testing.T) {
	// S₂ does NOT consume S₁'s output → F1 violated
	s1 := makeSkill("producer", nil, []string{"x"})
	s2 := makeSkill("consumer", []string{"y"}, []string{"result"})
	all := []model.SkillBehavior{s1, s2}
	agent := makeAgent("test", []string{"producer", "consumer"}, nil, []string{"result"})

	assert.False(t, CanFuse(s1, s2, agent, all))
}

func TestP11_Fusion_NoProduces(t *testing.T) {
	// S₁ produces nothing → cannot fuse
	s1 := makeSkill("empty", nil, nil)
	s2 := makeSkill("consumer", nil, []string{"result"})
	all := []model.SkillBehavior{s1, s2}
	agent := makeAgent("test", []string{"empty", "consumer"}, nil, []string{"result"})

	assert.False(t, CanFuse(s1, s2, agent, all))
}

// --- P12: Isolation ---

func TestP12_Isolation_DisjointConsumes(t *testing.T) {
	// Two skills with disjoint consumes in the same layer → isolated.
	// Build the graph and verify they share no data-flow edge.
	skills := []model.SkillBehavior{
		makeSkill("linter", []string{"source_code"}, []string{"lint_results"}),
		makeSkill("tester", []string{"test_suite"}, []string{"test_results"}),
	}
	agent := makeAgent("test",
		[]string{"linter", "tester"}, nil,
		[]string{"lint_results", "test_results"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	// Both should be in layer 0 (parallel)
	require.Equal(t, 1, len(layers))
	assert.Len(t, layers[0], 2)

	// No edge between them in either direction
	assert.False(t, g.Reachable("linter", "tester"))
	assert.False(t, g.Reachable("tester", "linter"))
}

func TestP12_Isolation_SharedConsumes_NotIsolated(t *testing.T) {
	// Two skills consuming the same type — they are parallel but share input.
	// Under context resolver, this breaks isolation (whitepaper caveat).
	s1 := makeSkill("linter", []string{"source_code"}, []string{"lint_results"})
	s2 := makeSkill("checker", []string{"source_code"}, []string{"type_errors"})

	shared := ConsumesOverlap(s1, s2)
	assert.Equal(t, []string{"source_code"}, shared,
		"shared consumes should be detected")
}

func TestP12_Isolation_ProducerConsumerNotIsolated(t *testing.T) {
	// S₁ produces what S₂ consumes — they are NOT isolated (data flows between them).
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"data"}),
		makeSkill("b", []string{"data"}, []string{"result"}),
	}
	agent := makeAgent("test", []string{"a", "b"}, nil, []string{"result"})
	g := BuildGraph(agent, skills)

	assert.True(t, g.Reachable("a", "b"), "producer reaches consumer — not isolated")
}

// --- P13: Environment Containment ---

func TestP13_Containment_AllLevels(t *testing.T) {
	// Exhaustive test of the permission lattice ordering.
	tests := []struct {
		child, parent model.AccessLevel
		contained     bool
	}{
		{model.AccessNone, model.AccessNone, true},
		{model.AccessNone, model.AccessFull, true},
		{model.AccessReadOnly, model.AccessReadWrite, true},
		{model.AccessReadWrite, model.AccessReadOnly, false},
		{model.AccessFull, model.AccessNone, false},
		{model.AccessFull, model.AccessFull, true},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.contained, AccessLevelContained(tt.child, tt.parent),
			"%s ⊆ %s should be %v", tt.child, tt.parent, tt.contained)
	}
}

func TestP13_Containment_NetworkLevels(t *testing.T) {
	tests := []struct {
		child, parent model.NetworkAccess
		contained     bool
	}{
		{model.NetworkNone, model.NetworkNone, true},
		{model.NetworkNone, model.NetworkFull, true},
		{model.NetworkAllowlist, model.NetworkFull, true},
		{model.NetworkFull, model.NetworkNone, false},
		{model.NetworkFull, model.NetworkAllowlist, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.contained, NetworkContained(tt.child, tt.parent),
			"%s ⊆ %s should be %v", tt.child, tt.parent, tt.contained)
	}
}

func TestP13_Containment_CIReviewer(t *testing.T) {
	// Verify every skill's permissions ⊆ max(skill permissions) in ci-reviewer.
	skills, _ := ciReviewerFixture()
	// Agent envelope must contain max(skill permissions).
	// tdd-runner has full filesystem access → agent needs full.
	agentFS := model.AccessFull
	agentNet := model.NetworkNone

	for _, s := range skills {
		assert.True(t, AccessLevelContained(s.Security.Filesystem, agentFS),
			"skill %q filesystem %s should be ⊆ %s", s.Skill, s.Security.Filesystem, agentFS)
		assert.True(t, NetworkContained(s.Security.Network, agentNet),
			"skill %q network %s should be ⊆ %s", s.Skill, s.Security.Network, agentNet)
	}
}

// --- P14: Parallel Independence ---

func TestP14_ParallelIndependence_CIReviewer(t *testing.T) {
	// Layer-0 skills in ci-reviewer are parallel.
	// P14: their write sets (produces) must be disjoint.
	skills, agent := ciReviewerFixture()
	g := BuildGraph(agent, skills)
	layers := g.Layers()
	require.GreaterOrEqual(t, len(layers), 1)

	skillMap := make(map[string]model.SkillBehavior)
	for _, s := range skills {
		skillMap[s.Skill] = s
	}

	allProduces := make(map[string]string) // type → skill name
	for _, sName := range layers[0] {
		s := skillMap[sName]
		for _, p := range s.Context.Produces {
			existing, dup := allProduces[p]
			assert.False(t, dup,
				"type %q produced by both %q and %q in same layer", p, existing, sName)
			allProduces[p] = sName
		}
	}
}

func TestP14_ParallelIndependence_Violation(t *testing.T) {
	// Two parallel skills producing the same type → P14 violated.
	s1 := makeSkill("a", nil, []string{"output"})
	s2 := makeSkill("b", nil, []string{"output"})
	agent := makeAgent("test", []string{"a", "b"}, nil, []string{"output"})

	g := BuildGraph(agent, skills(s1, s2))
	layers := g.Layers()
	require.Equal(t, 1, len(layers))

	// Both in layer 0 but produce same type — violation detected
	seen := make(map[string]bool)
	violation := false
	for _, sName := range layers[0] {
		for _, sk := range skills(s1, s2) {
			if sk.Skill == sName {
				for _, p := range sk.Context.Produces {
					if seen[p] {
						violation = true
					}
					seen[p] = true
				}
			}
		}
	}
	assert.True(t, violation, "should detect duplicate produces in same layer")

	// Also verify via DisjointProduces helper
	assert.False(t, DisjointProduces([]model.SkillBehavior{s1, s2}))
}

// --- P15: Conflict-Free Merge ---

func TestP15_ConflictFreeMerge_Disjoint(t *testing.T) {
	// Disjoint write sets → merge is conflict-free.
	s1 := makeSkill("linter", []string{"source_code"}, []string{"lint_results"})
	s2 := makeSkill("tester", []string{"source_code"}, []string{"test_results"})

	assert.True(t, DisjointProduces([]model.SkillBehavior{s1, s2}),
		"disjoint produces → conflict-free merge")
}

func TestP15_ConflictFreeMerge_Conflict(t *testing.T) {
	// Overlapping write sets → merge has conflicts.
	s1 := makeSkill("writer1", nil, []string{"shared_output"})
	s2 := makeSkill("writer2", nil, []string{"shared_output"})

	assert.False(t, DisjointProduces([]model.SkillBehavior{s1, s2}),
		"overlapping produces → merge conflict")
}

func TestP15_ConflictFreeMerge_CIReviewerLayer0(t *testing.T) {
	// In ci-reviewer, all layer-0 skills have disjoint produces.
	skills, agent := ciReviewerFixture()
	g := BuildGraph(agent, skills)
	layers := g.Layers()
	require.GreaterOrEqual(t, len(layers), 1)

	skillMap := make(map[string]model.SkillBehavior)
	for _, s := range skills {
		skillMap[s.Skill] = s
	}

	var layer0Skills []model.SkillBehavior
	for _, name := range layers[0] {
		layer0Skills = append(layer0Skills, skillMap[name])
	}
	assert.True(t, DisjointProduces(layer0Skills),
		"ci-reviewer layer 0 should have disjoint produces")
}

// --- Test fixtures ---

func ciReviewerFixture() ([]model.SkillBehavior, model.AgentComposition) {
	skills := []model.SkillBehavior{
		makeSkillWithSecurity("ts-linter", []string{"file_tree", "source_code"}, []string{"lint_results"}, model.AccessReadOnly, model.NetworkNone),
		makeSkillWithSecurity("type-checker", []string{"file_tree", "source_code"}, []string{"type_errors"}, model.AccessReadOnly, model.NetworkNone),
		makeSkillWithSecurity("tdd-runner", []string{"file_tree", "source_code"}, []string{"test_results"}, model.AccessFull, model.NetworkNone),
		makeSkillWithSecurity("coverage-reporter", []string{"file_tree", "source_code"}, []string{"coverage_report"}, model.AccessReadOnly, model.NetworkNone),
		makeSkillWithSecurity("review-commenter", []string{"git_diff", "test_results", "lint_results"}, []string{"review_comments"}, model.AccessReadOnly, model.NetworkNone),
		makeSkillWithSecurity("risk-scorer", []string{"git_diff", "test_results", "lint_results"}, []string{"risk_score"}, model.AccessReadOnly, model.NetworkNone),
	}
	agent := makeAgent("ci-reviewer",
		[]string{"ts-linter", "type-checker", "tdd-runner", "coverage-reporter", "review-commenter", "risk-scorer"},
		nil,
		[]string{"lint_results", "type_errors", "test_results", "coverage_report", "review_comments", "risk_score"},
	)
	return skills, agent
}

func skills(ss ...model.SkillBehavior) []model.SkillBehavior {
	return ss
}
