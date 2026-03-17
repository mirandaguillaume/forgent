// pkg/dag/dag.go
package dag

import (
	"context"
	"fmt"
	"time"
)

// NodeKind distinguishes execution semantics.
type NodeKind string

const (
	KindTask   NodeKind = "task"
	KindRouter NodeKind = "router"
	KindMerge  NodeKind = "merge"
)

// Node is a unit of work in the DAG.
type Node struct {
	ID         string
	Kind       NodeKind
	Consumes   []string
	Produces   []string
	Run        func(ctx context.Context, inputs map[string]any) (map[string]any, error)
	MaxRetries int
	Timeout    time.Duration
}

// DAG is a directed acyclic graph auto-wired by type-matching edges.
type DAG struct {
	nodes map[string]*Node
	adj   map[string][]string
	rev   map[string][]string
}

// New creates a DAG and auto-wires edges from Produces/Consumes type matching.
func New(nodes ...*Node) (*DAG, error) {
	d := &DAG{
		nodes: make(map[string]*Node, len(nodes)),
		adj:   make(map[string][]string, len(nodes)),
		rev:   make(map[string][]string, len(nodes)),
	}

	for _, n := range nodes {
		if _, exists := d.nodes[n.ID]; exists {
			return nil, fmt.Errorf("dag: duplicate node ID %q", n.ID)
		}
		if n.Kind == "" {
			n.Kind = KindTask
		}
		d.nodes[n.ID] = n
		d.adj[n.ID] = nil
		d.rev[n.ID] = nil
	}

	producer := make(map[string]string, len(nodes))
	for _, n := range nodes {
		for _, p := range n.Produces {
			producer[p] = n.ID
		}
	}

	for _, n := range nodes {
		for _, c := range n.Consumes {
			if src, ok := producer[c]; ok && src != n.ID {
				d.addEdge(src, n.ID)
			}
		}
	}

	return d, nil
}

// AddEdge adds a manual directed edge from → to.
func (d *DAG) AddEdge(from, to string) error {
	if _, ok := d.nodes[from]; !ok {
		return fmt.Errorf("dag: unknown node %q", from)
	}
	if _, ok := d.nodes[to]; !ok {
		return fmt.Errorf("dag: unknown node %q", to)
	}
	d.addEdge(from, to)
	return nil
}

// RemoveEdge removes the edge from → to.
func (d *DAG) RemoveEdge(from, to string) error {
	neighbors := d.adj[from]
	for i, n := range neighbors {
		if n == to {
			d.adj[from] = append(neighbors[:i], neighbors[i+1:]...)
			break
		}
	}
	rev := d.rev[to]
	for i, n := range rev {
		if n == from {
			d.rev[to] = append(rev[:i], rev[i+1:]...)
			break
		}
	}
	return nil
}

// Downstream returns the node IDs that nodeID points to.
func (d *DAG) Downstream(nodeID string) []string { return d.adj[nodeID] }

// Upstream returns the node IDs that point to nodeID.
func (d *DAG) Upstream(nodeID string) []string { return d.rev[nodeID] }

// Nodes returns all node IDs.
func (d *DAG) Nodes() []string {
	ids := make([]string, 0, len(d.nodes))
	for id := range d.nodes {
		ids = append(ids, id)
	}
	return ids
}

// Node returns the node with the given ID, or nil.
func (d *DAG) Node(id string) *Node { return d.nodes[id] }

func (d *DAG) addEdge(from, to string) {
	for _, n := range d.adj[from] {
		if n == to {
			return
		}
	}
	d.adj[from] = append(d.adj[from], to)
	d.rev[to] = append(d.rev[to], from)
}
