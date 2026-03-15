// Package formal implements the DAG abstraction from §4 of the whitepaper
// and tests formal properties P10-P15.
package formal

import (
	"github.com/mirandaguillaume/forgent/pkg/model"
)

const (
	Top    = "⊤" // virtual source — agent-level inputs
	Bottom = "⊥" // virtual sink — agent-level outputs
)

// Graph represents the directed acyclic graph G(A) from the whitepaper.
// Nodes are skill names plus the virtual source (⊤) and sink (⊥).
// Edges represent data flow: (S₁, S₂) means S₁ produces a type that S₂ consumes.
type Graph struct {
	Nodes  []string
	Adj    map[string][]string // outgoing edges
	RevAdj map[string][]string // incoming edges (reverse)
}

// BuildGraph constructs G(A) from an agent composition and its resolved skills.
//
// Edge rules:
//   - ⊤ → Sᵢ if Sᵢ consumes a type in the agent's external inputs (Cₐ)
//     or if Sᵢ consumes a type not produced by any other skill (source skill)
//   - Sᵢ → Sⱼ if P(Sᵢ) ∈ C(Sⱼ) (Sⱼ consumes what Sᵢ produces)
//   - Sᵢ → ⊥ if P(Sᵢ) ∈ Pₐ (Sᵢ produces an agent-level output)
func BuildGraph(agent model.AgentComposition, skills []model.SkillBehavior) *Graph {
	g := &Graph{
		Adj:    make(map[string][]string),
		RevAdj: make(map[string][]string),
	}

	// Build producer index: type → skill name
	producer := make(map[string]string)
	skillMap := make(map[string]model.SkillBehavior)
	for _, s := range skills {
		skillMap[s.Skill] = s
		for _, p := range s.Context.Produces {
			producer[p] = s.Skill
		}
	}

	// Agent-level outputs set
	agentOutputs := toSet(agent.Produces)

	// Add nodes
	g.Nodes = append(g.Nodes, Top)
	for _, s := range skills {
		g.Nodes = append(g.Nodes, s.Skill)
	}
	g.Nodes = append(g.Nodes, Bottom)

	// Add edges
	for _, s := range skills {
		for _, c := range s.Context.Consumes {
			if prod, ok := producer[c]; ok {
				g.addEdge(prod, s.Skill)
			} else {
				// Consumed type has no internal producer → comes from ⊤
				g.addEdge(Top, s.Skill)
			}
		}
		if len(s.Context.Consumes) == 0 {
			// Source skill with no inputs → connect from ⊤
			g.addEdge(Top, s.Skill)
		}

		for _, p := range s.Context.Produces {
			if agentOutputs[p] {
				g.addEdge(s.Skill, Bottom)
			}
		}
	}

	return g
}

func (g *Graph) addEdge(from, to string) {
	// Avoid duplicate edges
	for _, n := range g.Adj[from] {
		if n == to {
			return
		}
	}
	g.Adj[from] = append(g.Adj[from], to)
	g.RevAdj[to] = append(g.RevAdj[to], from)
}

// Reachable returns true if `to` is reachable from `from` via BFS.
func (g *Graph) Reachable(from, to string) bool {
	visited := make(map[string]bool)
	queue := []string{from}
	visited[from] = true

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			return true
		}
		for _, next := range g.Adj[cur] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

// Layers returns the topological layer decomposition (Corollary 3.1).
// Layer 0 contains skills reachable directly from ⊤.
// Layer i contains skills whose longest path from ⊤ is i.
// ⊤ and ⊥ are excluded from the layers.
func (g *Graph) Layers() [][]string {
	// Compute longest path distance from ⊤ for each node
	dist := make(map[string]int)
	for _, n := range g.Nodes {
		dist[n] = -1
	}
	dist[Top] = 0

	// Topological order via Kahn's algorithm
	inDegree := make(map[string]int)
	for _, n := range g.Nodes {
		inDegree[n] = 0
	}
	for _, neighbors := range g.Adj {
		for _, n := range neighbors {
			inDegree[n]++
		}
	}

	queue := []string{}
	for _, n := range g.Nodes {
		if inDegree[n] == 0 {
			queue = append(queue, n)
		}
	}

	var topoOrder []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		topoOrder = append(topoOrder, cur)
		for _, next := range g.Adj[cur] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// Longest path from ⊤
	for _, n := range topoOrder {
		if dist[n] < 0 {
			continue
		}
		for _, next := range g.Adj[n] {
			if dist[n]+1 > dist[next] {
				dist[next] = dist[n] + 1
			}
		}
	}

	// Group skill nodes by layer (exclude ⊤ and ⊥)
	maxLayer := 0
	for _, n := range g.Nodes {
		if n == Top || n == Bottom {
			continue
		}
		if dist[n] > maxLayer {
			maxLayer = dist[n]
		}
	}

	layers := make([][]string, maxLayer)
	for _, n := range g.Nodes {
		if n == Top || n == Bottom {
			continue
		}
		if dist[n] > 0 {
			layers[dist[n]-1] = append(layers[dist[n]-1], n) // shift by 1 since ⊤ is layer 0
		}
	}

	// Remove empty trailing layers
	for len(layers) > 0 && len(layers[len(layers)-1]) == 0 {
		layers = layers[:len(layers)-1]
	}

	return layers
}

// Width returns the Dilworth width — the maximum layer size (Corollary 3.2).
func (g *Graph) Width() int {
	layers := g.Layers()
	max := 0
	for _, l := range layers {
		if len(l) > max {
			max = len(l)
		}
	}
	return max
}

// CanFuse checks the fusion conditions F1-F3 from Proposition 11.
// F1: P(S₁) ∈ C(S₂)
// F2: P(S₁) ∉ C(Sⱼ) for all Sⱼ ≠ S₂
// F3: P(S₁) ∉ Pₐ
func CanFuse(s1, s2 model.SkillBehavior, agent model.AgentComposition, allSkills []model.SkillBehavior) bool {
	if len(s1.Context.Produces) == 0 {
		return false
	}
	output := s1.Context.Produces[0]

	// F1: S₂ consumes S₁'s output
	if !contains(s2.Context.Consumes, output) {
		return false
	}

	// F2: No other skill consumes S₁'s output
	for _, s := range allSkills {
		if s.Skill == s1.Skill || s.Skill == s2.Skill {
			continue
		}
		if contains(s.Context.Consumes, output) {
			return false
		}
	}

	// F3: S₁'s output is not an agent-level output
	if contains(agent.Produces, output) {
		return false
	}

	return true
}

// ConsumesOverlap returns the types consumed by both skills.
func ConsumesOverlap(s1, s2 model.SkillBehavior) []string {
	set := toSet(s1.Context.Consumes)
	var overlap []string
	for _, c := range s2.Context.Consumes {
		if set[c] {
			overlap = append(overlap, c)
		}
	}
	return overlap
}

// DisjointProduces returns true if no two skills produce the same type.
func DisjointProduces(skills []model.SkillBehavior) bool {
	seen := make(map[string]bool)
	for _, s := range skills {
		for _, p := range s.Context.Produces {
			if seen[p] {
				return false
			}
			seen[p] = true
		}
	}
	return true
}

// AccessLevelContained returns true if child ⊆ parent in the permission lattice.
func AccessLevelContained(child, parent model.AccessLevel) bool {
	return accessOrder[child] <= accessOrder[parent]
}

// NetworkContained returns true if child ⊆ parent in the permission lattice.
func NetworkContained(child, parent model.NetworkAccess) bool {
	return networkOrder[child] <= networkOrder[parent]
}

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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func toSet(slice []string) map[string]bool {
	m := make(map[string]bool, len(slice))
	for _, s := range slice {
		m[s] = true
	}
	return m
}
