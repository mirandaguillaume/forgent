package dag_test

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/dag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func node(id string, consumes, produces []string) *dag.Node {
	return &dag.Node{ID: id, Consumes: consumes, Produces: produces}
}

func TestNew_Pipeline(t *testing.T) {
	a := node("a", nil, []string{"x"})
	b := node("b", []string{"x"}, []string{"y"})
	c := node("c", []string{"y"}, []string{"z"})

	d, err := dag.New(a, b, c)
	require.NoError(t, err)

	assert.Contains(t, d.Downstream("a"), "b")
	assert.Contains(t, d.Downstream("b"), "c")
	assert.Empty(t, d.Upstream("a"))
}

func TestNew_Diamond(t *testing.T) {
	a := node("a", nil, []string{"x"})
	b := node("b", []string{"x"}, []string{"y"})
	c := node("c", []string{"x"}, []string{"z"})
	d := node("d", []string{"y", "z"}, []string{"out"})

	g, err := dag.New(a, b, c, d)
	require.NoError(t, err)

	assert.Contains(t, g.Downstream("a"), "b")
	assert.Contains(t, g.Downstream("a"), "c")
	assert.Contains(t, g.Downstream("b"), "d")
	assert.Contains(t, g.Downstream("c"), "d")
}

func TestNew_DuplicateID(t *testing.T) {
	_, err := dag.New(
		node("a", nil, []string{"x"}),
		node("a", []string{"x"}, []string{"y"}),
	)
	assert.Error(t, err)
}

func TestAddRemoveEdge(t *testing.T) {
	a := node("a", nil, []string{"x"})
	b := node("b", nil, []string{"y"})
	g, err := dag.New(a, b)
	require.NoError(t, err)

	assert.Empty(t, g.Downstream("a"))

	require.NoError(t, g.AddEdge("a", "b"))
	assert.Contains(t, g.Downstream("a"), "b")

	require.NoError(t, g.RemoveEdge("a", "b"))
	assert.Empty(t, g.Downstream("a"))
}
