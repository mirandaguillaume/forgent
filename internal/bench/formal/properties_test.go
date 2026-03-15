package formal

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- P10: Reachability ---

func TestP10_Reachability(t *testing.T) {
	// Every skill is reachable from ⊤, and ⊥ is reachable from every skill.
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
	assert.True(t, g.Reachable("c", Bottom), "⊥ should be reachable from c")
}

func TestP10_Reachability_SourceSkill(t *testing.T) {
	// A skill with C=∅ (no consumes) should still be reachable from ⊤.
	skills := []model.SkillBehavior{
		makeSkill("source", nil, []string{"data"}),
		makeSkill("sink", []string{"data"}, []string{"result"}),
	}
	agent := makeAgent("test", []string{"source", "sink"}, nil, []string{"result"})

	g := BuildGraph(agent, skills)
	assert.True(t, g.Reachable(Top, "source"), "source skill should be reachable from ⊤")
	assert.True(t, g.Reachable(Top, "sink"), "sink skill should be reachable from ⊤")
}

// --- Corollary 3.1: Layer Decomposition ---

func TestCor31_LayerDecomposition(t *testing.T) {
	// Layers form a unique partition; intra-layer skills are independent.
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
	layers := g.Layers()

	// Verify partition: every skill appears exactly once
	seen := make(map[string]bool)
	for _, layer := range layers {
		for _, s := range layer {
			assert.False(t, seen[s], "skill %q appears in multiple layers", s)
			seen[s] = true
		}
	}
	for _, s := range skills {
		assert.True(t, seen[s.Skill], "skill %q not in any layer", s.Skill)
	}

	// Verify intra-layer independence: no edge between skills in the same layer
	for _, layer := range layers {
		layerSet := toSet(layer)
		for _, s := range layer {
			for _, neighbor := range g.Adj[s] {
				assert.False(t, layerSet[neighbor],
					"intra-layer edge %s → %s violates independence", s, neighbor)
			}
		}
	}
}

// --- Corollary 3.2: Dilworth Width ---

func TestCor32_DilworthWidth(t *testing.T) {
	// Width = max layer size.
	skills := []model.SkillBehavior{
		makeSkill("a", nil, []string{"x"}),
		makeSkill("b", nil, []string{"y"}),
		makeSkill("c", []string{"x", "y"}, []string{"z"}),
	}
	agent := makeAgent("test", []string{"a", "b", "c"}, nil, []string{"z"})

	g := BuildGraph(agent, skills)
	layers := g.Layers()

	maxSize := 0
	for _, l := range layers {
		if len(l) > maxSize {
			maxSize = len(l)
		}
	}
	assert.Equal(t, maxSize, g.Width())
	assert.Equal(t, 2, g.Width(), "a and b are parallel → width 2")
}

// --- P11: Skill Fusion ---

func TestP11_Fusion(t *testing.T) {
	// S₁ produces x, only S₂ consumes x, x is not an agent output → fusible.
	s1 := makeSkill("producer", nil, []string{"intermediate"})
	s2 := makeSkill("consumer", []string{"intermediate"}, []string{"result"})
	agent := makeAgent("test",
		[]string{"producer", "consumer"}, nil, []string{"result"})

	assert.True(t, CanFuse(s1, s2, agent, []model.SkillBehavior{s1, s2}),
		"exclusively-connected pair should be fusible")
}

func TestP11_Fusion_SharedConsumer(t *testing.T) {
	// S₁ produces x, both S₂ and S₃ consume x → NOT fusible (F2 violated).
	s1 := makeSkill("producer", nil, []string{"intermediate"})
	s2 := makeSkill("consumer1", []string{"intermediate"}, []string{"r1"})
	s3 := makeSkill("consumer2", []string{"intermediate"}, []string{"r2"})
	agent := makeAgent("test",
		[]string{"producer", "consumer1", "consumer2"}, nil, []string{"r1", "r2"})

	assert.False(t, CanFuse(s1, s2, agent, []model.SkillBehavior{s1, s2, s3}),
		"shared consumer violates F2 → not fusible")
}

func TestP11_Fusion_AgentOutput(t *testing.T) {
	// S₁ produces x which is an agent output → NOT fusible (F3 violated).
	s1 := makeSkill("producer", nil, []string{"intermediate"})
	s2 := makeSkill("consumer", []string{"intermediate"}, []string{"result"})
	agent := makeAgent("test",
		[]string{"producer", "consumer"}, nil, []string{"intermediate", "result"})

	assert.False(t, CanFuse(s1, s2, agent, []model.SkillBehavior{s1, s2}),
		"agent output violates F3 → not fusible")
}

// --- P12: Isolation ---

func TestP12_Isolation(t *testing.T) {
	// Disjoint consumes → no data sharing between skills.
	s1 := makeSkill("linter", []string{"source_code"}, []string{"lint_results"})
	s2 := makeSkill("tester", []string{"test_suite"}, []string{"test_results"})

	// P12: C(S₁) ∩ C(S₂) = ∅ → they access disjoint data
	c1 := toSet(s1.Context.Consumes)
	c2 := toSet(s2.Context.Consumes)

	for k := range c1 {
		assert.False(t, c2[k],
			"type %q in both consumes sets violates isolation", k)
	}
}

func TestP12_Isolation_SharedInput(t *testing.T) {
	// Shared input → isolation caveat applies.
	s1 := makeSkill("linter", []string{"source_code"}, []string{"lint_results"})
	s2 := makeSkill("checker", []string{"source_code"}, []string{"type_errors"})

	c1 := toSet(s1.Context.Consumes)
	c2 := toSet(s2.Context.Consumes)

	hasOverlap := false
	for k := range c1 {
		if c2[k] {
			hasOverlap = true
			break
		}
	}
	assert.True(t, hasOverlap,
		"shared input means isolation depends on resolver type (context vs file)")
}

// --- P13: Environment Containment ---

func TestP13_Containment(t *testing.T) {
	// Skill permissions ⊆ agent permissions.
	// Here: skill has read-only fs, no network. Agent has read-write fs, full network.
	skill := makeSkillWithSecurity("linter",
		[]string{"source_code"}, []string{"lint_results"},
		model.AccessReadOnly, model.NetworkNone)
	agentSecurity := model.SecurityFacet{
		Filesystem: model.AccessReadWrite,
		Network:    model.NetworkFull,
	}

	assert.True(t, accessLevelContained(skill.Security.Filesystem, agentSecurity.Filesystem),
		"skill filesystem ⊆ agent filesystem")
	assert.True(t, networkContained(skill.Security.Network, agentSecurity.Network),
		"skill network ⊆ agent network")
}

func TestP13_Containment_Violation(t *testing.T) {
	// Skill has full fs, agent has read-only → violation.
	skill := makeSkillWithSecurity("writer",
		nil, []string{"output"},
		model.AccessFull, model.NetworkFull)
	agentSecurity := model.SecurityFacet{
		Filesystem: model.AccessReadOnly,
		Network:    model.NetworkNone,
	}

	assert.False(t, accessLevelContained(skill.Security.Filesystem, agentSecurity.Filesystem),
		"skill filesystem should NOT be contained in agent filesystem")
	assert.False(t, networkContained(skill.Security.Network, agentSecurity.Network),
		"skill network should NOT be contained in agent network")
}

// --- P14: Parallel Independence ---

func TestP14_ParallelIndependence(t *testing.T) {
	// Parallel skills → disjoint produces (write sets).
	s1 := makeSkill("linter", []string{"source_code"}, []string{"lint_results"})
	s2 := makeSkill("tester", []string{"source_code"}, []string{"test_results"})

	p1 := toSet(s1.Context.Produces)
	p2 := toSet(s2.Context.Produces)

	for k := range p1 {
		assert.False(t, p2[k],
			"type %q produced by both parallel skills violates P14", k)
	}
}

func TestP14_ParallelIndependence_CIReviewer(t *testing.T) {
	// In ci-reviewer, layer-0 skills (linter, checker, runner, reporter) are parallel.
	// Their produces must be disjoint.
	skills := []model.SkillBehavior{
		makeSkill("ts-linter", []string{"file_tree", "source_code"}, []string{"lint_results"}),
		makeSkill("type-checker", []string{"file_tree", "source_code"}, []string{"type_errors"}),
		makeSkill("tdd-runner", []string{"file_tree", "source_code"}, []string{"test_results"}),
		makeSkill("coverage-reporter", []string{"file_tree", "source_code"}, []string{"coverage_report"}),
	}
	agent := makeAgent("ci-reviewer",
		[]string{"ts-linter", "type-checker", "tdd-runner", "coverage-reporter"},
		nil,
		[]string{"lint_results", "type_errors", "test_results", "coverage_report"},
	)

	g := BuildGraph(agent, skills)
	layers := g.Layers()
	require.GreaterOrEqual(t, len(layers), 1)

	// Check all skills in layer 0 have disjoint produces
	layer0Skills := layers[0]
	allProduces := make(map[string]string) // type → producing skill
	for _, sName := range layer0Skills {
		for _, skill := range skills {
			if skill.Skill == sName {
				for _, p := range skill.Context.Produces {
					existing, dup := allProduces[p]
					assert.False(t, dup,
						"type %q produced by both %q and %q in same layer", p, existing, sName)
					allProduces[p] = sName
				}
			}
		}
	}
}

// --- P15: Conflict-Free Merge ---

func TestP15_ConflictFreeMerge(t *testing.T) {
	// Disjoint write sets → commutative merge.
	s1 := makeSkill("linter", []string{"source_code"}, []string{"lint_results"})
	s2 := makeSkill("tester", []string{"source_code"}, []string{"test_results"})

	// Write sets = produces
	w1 := toSet(s1.Context.Produces)
	w2 := toSet(s2.Context.Produces)

	disjoint := true
	for k := range w1 {
		if w2[k] {
			disjoint = false
		}
	}
	assert.True(t, disjoint, "disjoint write sets")

	// Commutativity: merge(W₁, W₂) = merge(W₂, W₁)
	// Simulated: combining outputs in either order produces the same set
	merged12 := make(map[string]bool)
	merged21 := make(map[string]bool)
	for k := range w1 {
		merged12[k] = true
		merged21[k] = true
	}
	for k := range w2 {
		merged12[k] = true
		merged21[k] = true
	}
	assert.Equal(t, merged12, merged21, "merge should be commutative")
}

// --- Helper: permission ordering ---

var accessOrder = map[model.AccessLevel]int{
	model.AccessNone:      0,
	model.AccessReadOnly:  1,
	model.AccessReadWrite: 2,
	model.AccessFull:      3,
}

var networkOrder = map[model.NetworkAccess]int{
	model.NetworkNone:      0,
	model.NetworkAllowlist: 1,
	model.NetworkFull:      2,
}

// accessLevelContained returns true if child ⊆ parent in the permission lattice.
func accessLevelContained(child, parent model.AccessLevel) bool {
	return accessOrder[child] <= accessOrder[parent]
}

// networkContained returns true if child ⊆ parent in the permission lattice.
func networkContained(child, parent model.NetworkAccess) bool {
	return networkOrder[child] <= networkOrder[parent]
}
