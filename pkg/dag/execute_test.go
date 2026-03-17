package dag_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/mirandaguillaume/forgent/pkg/dag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runNode is a test helper: creates a node whose Run sets each produced type to the value from extra,
// or to "<id>_output" if not in extra.
func runNode(id string, consumes, produces []string, extra map[string]any) *dag.Node {
	n := &dag.Node{ID: id, Consumes: consumes, Produces: produces}
	n.Run = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		out := make(map[string]any, len(produces))
		for _, p := range produces {
			if v, ok := extra[p]; ok {
				out[p] = v
			} else {
				out[p] = id + "_output"
			}
		}
		return out, nil
	}
	return n
}

func TestExecute_Pipeline(t *testing.T) {
	var order []string
	mkNode := func(id string, consumes, produces []string) *dag.Node {
		n := &dag.Node{ID: id, Consumes: consumes, Produces: produces}
		n.Run = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			order = append(order, id)
			out := map[string]any{}
			for _, p := range produces {
				out[p] = id
			}
			return out, nil
		}
		return n
	}

	d, _ := dag.New(
		mkNode("a", nil, []string{"x"}),
		mkNode("b", []string{"x"}, []string{"y"}),
		mkNode("c", []string{"y"}, []string{"z"}),
	)
	results, err := d.Execute(context.Background())
	require.NoError(t, err)

	require.Equal(t, []string{"a", "b", "c"}, order)
	assert.Equal(t, "c", results["z"])
}

func TestExecute_Diamond_Parallel(t *testing.T) {
	d, _ := dag.New(
		runNode("a", nil, []string{"x"}, map[string]any{"x": "from-a"}),
		runNode("b", []string{"x"}, []string{"y"}, map[string]any{"y": "from-b"}),
		runNode("c", []string{"x"}, []string{"z"}, map[string]any{"z": "from-c"}),
		runNode("dd", []string{"y", "z"}, []string{"out"}, map[string]any{"out": "merged"}),
	)
	results, err := d.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "merged", results["out"])
}

func TestExecute_WithInitialInputs(t *testing.T) {
	b := &dag.Node{
		ID:       "b",
		Consumes: []string{"req"},
		Produces: []string{"resp"},
		Run: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			v, _ := inputs["req"].(string)
			return map[string]any{"resp": "got:" + v}, nil
		},
	}
	d, _ := dag.New(b)
	results, err := d.Execute(context.Background(),
		dag.WithInputs(map[string]any{"req": "hello"}),
	)
	require.NoError(t, err)
	assert.Equal(t, "got:hello", results["resp"])
}

func TestExecute_Router_S3(t *testing.T) {
	router := &dag.Node{
		ID:       "router",
		Kind:     dag.KindRouter,
		Consumes: []string{"score"},
		Produces: []string{"__route"},
		Run: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			score, _ := inputs["score"].(int)
			if score > 50 {
				return map[string]any{"__route": "left"}, nil
			}
			return map[string]any{"__route": "right"}, nil
		},
	}
	left := runNode("left", nil, []string{"left_out"}, map[string]any{"left_out": "HIGH"})
	right := runNode("right", nil, []string{"right_out"}, map[string]any{"right_out": "LOW"})

	d, _ := dag.New(router, left, right)
	require.NoError(t, d.AddEdge("router", "left"))
	require.NoError(t, d.AddEdge("router", "right"))

	results, err := d.Execute(context.Background(),
		dag.WithInputs(map[string]any{"score": 90}),
	)
	require.NoError(t, err)
	assert.Equal(t, "HIGH", results["left_out"])
	assert.Nil(t, results["right_out"], "right branch should be skipped")
}

func TestExecute_FailFast(t *testing.T) {
	fail := &dag.Node{
		ID:       "fail",
		Produces: []string{"x"},
		Run: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			return nil, errors.New("boom")
		},
	}
	next := runNode("next", []string{"x"}, []string{"y"}, nil)
	d, _ := dag.New(fail, next)

	_, err := d.Execute(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestExecute_OnNodeComplete(t *testing.T) {
	var completed []string
	var mu sync.Mutex

	d, _ := dag.New(
		runNode("a", nil, []string{"x"}, nil),
		runNode("b", []string{"x"}, []string{"y"}, nil),
	)
	_, err := d.Execute(context.Background(),
		dag.WithOnNodeComplete(func(nodeID string) {
			mu.Lock()
			completed = append(completed, nodeID)
			mu.Unlock()
		}),
	)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "b"}, completed)
}

func TestExecute_NilRun_Skipped(t *testing.T) {
	// Nodes with nil Run should be skipped gracefully (used by formal.Graph analysis nodes)
	n := &dag.Node{ID: "n", Produces: []string{"x"}} // Run is nil
	d, _ := dag.New(n)
	results, err := d.Execute(context.Background())
	require.NoError(t, err)
	_ = results // nil Run produces no output
}

func TestExecute_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	slow := &dag.Node{
		ID:       "slow",
		Produces: []string{"x"},
		Run: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			select {
			case <-time.After(1 * time.Second):
				return map[string]any{"x": "done"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	d, _ := dag.New(slow)
	_, err := d.Execute(ctx)
	assert.Error(t, err)
}
