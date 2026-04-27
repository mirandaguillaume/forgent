package qualitygate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mirandaguillaume/forgent/pkg/dag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// handler is a test helper that returns fixed outputs.
func handler(outputs map[string]any) dag.MiddlewareFunc {
	return func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		return outputs, nil
	}
}

// failHandler returns a fixed error.
func failHandler(err error) dag.MiddlewareFunc {
	return func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		return nil, err
	}
}

// ─── extractMarkers tests ──────────────────────────────────────────────────

func TestExtractMarkers_Empty(t *testing.T) {
	assert.Nil(t, extractMarkers(""))
}

func TestExtractMarkers_Heading(t *testing.T) {
	markers := extractMarkers("## Bugs\n### Details")
	assert.Equal(t, []string{"## bugs", "### details"}, markers)
}

func TestExtractMarkers_BoldListItem(t *testing.T) {
	markers := extractMarkers("- **location**: description")
	assert.Equal(t, []string{"- **"}, markers)
}

func TestExtractMarkers_PlainListItem(t *testing.T) {
	markers := extractMarkers("- simple item")
	assert.Equal(t, []string{"- "}, markers)
}

func TestExtractMarkers_BoldItemTakesPrecedenceOverPlain(t *testing.T) {
	markers := extractMarkers("- **bold**: first\n- plain item")
	// "- **" is already recorded; "- " is a superset, not added separately
	assert.Equal(t, []string{"- **"}, markers)
}

func TestExtractMarkers_AsteriskListItem(t *testing.T) {
	markers := extractMarkers("* item")
	assert.Equal(t, []string{"- "}, markers)
}

func TestExtractMarkers_Deduplicated(t *testing.T) {
	markers := extractMarkers("## Bugs\n## Bugs\n- **a**: x\n- **b**: y")
	assert.Equal(t, []string{"## bugs", "- **"}, markers)
}

func TestExtractMarkers_FullTemplate(t *testing.T) {
	tmpl := "## Bugs\n- **location**: description of bug"
	markers := extractMarkers(tmpl)
	assert.Equal(t, []string{"## bugs", "- **"}, markers)
}

// ─── Policy passthrough tests ──────────────────────────────────────────────

func TestNilPolicy_Passthrough(t *testing.T) {
	mw := QualityGateMiddleware(nil)
	out := map[string]any{"result": "hello world"}
	got, err := mw(context.Background(), nil, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

func TestEmptyTemplates_Passthrough(t *testing.T) {
	mw := QualityGateMiddleware(&QualityGatePolicy{})
	out := map[string]any{"result": "hello world"}
	got, err := mw(context.Background(), nil, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

func TestNoMarkersInTemplate_Passthrough(t *testing.T) {
	// A template that produces no markers (plain prose) → pass-through
	mw := QualityGateMiddleware(&QualityGatePolicy{
		ProducesTemplate: "Just some prose with no headings or list items.",
	})
	out := map[string]any{"result": "anything goes"}
	got, err := mw(context.Background(), nil, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

// ─── ProducesTemplate tests ────────────────────────────────────────────────

func TestProducesTemplate_Pass(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Bugs\n- **location**: description",
	}
	mw := QualityGateMiddleware(policy)

	out := map[string]any{
		"bugs": "## Bugs\n- **src/main.go:12**: off-by-one error in loop boundary",
	}
	got, err := mw(context.Background(), nil, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

func TestProducesTemplate_Fail_MissingHeading(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Bugs\n- **location**: description",
	}
	mw := QualityGateMiddleware(policy)

	out := map[string]any{
		"bugs": "I found a bug at line 12. The loop counter overflows.",
	}
	_, err := mw(context.Background(), nil, handler(out))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quality gate")
	assert.Contains(t, err.Error(), "## bugs")
}

func TestProducesTemplate_Fail_MissingListItem(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Bugs\n- **location**: description",
	}
	mw := QualityGateMiddleware(policy)

	// Has the heading but no bold list items
	out := map[string]any{
		"bugs": "## Bugs\nThere are no bugs in this code.",
	}
	_, err := mw(context.Background(), nil, handler(out))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quality gate")
}

func TestProducesTemplate_CaseInsensitive(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Bugs",
	}
	mw := QualityGateMiddleware(policy)

	// Heading with different casing
	out := map[string]any{"result": "## BUGS\nsome findings"}
	got, err := mw(context.Background(), nil, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

func TestProducesTemplate_MultipleOutputValues(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Summary",
	}
	mw := QualityGateMiddleware(policy)

	// Marker appears in second output value
	out := map[string]any{
		"part1": "some analysis",
		"part2": "## Summary\nAll done.",
	}
	got, err := mw(context.Background(), nil, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

// ─── ConsumesTemplate tests ────────────────────────────────────────────────

func TestConsumesTemplate_Pass(t *testing.T) {
	policy := &QualityGatePolicy{
		ConsumesTemplate: "## Bugs\n- **location**: description",
	}
	mw := QualityGateMiddleware(policy)

	inputs := map[string]any{
		"bugs": "## Bugs\n- **src/main.go**: buffer overflow",
	}
	out := map[string]any{"fixed": "patch applied"}
	got, err := mw(context.Background(), inputs, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

func TestConsumesTemplate_Fail(t *testing.T) {
	policy := &QualityGatePolicy{
		ConsumesTemplate: "## Bugs\n- **location**: description",
	}
	mw := QualityGateMiddleware(policy)

	// Input is missing the expected structure
	inputs := map[string]any{
		"bugs": "here are some issues i found",
	}
	out := map[string]any{"fixed": "whatever"}
	_, err := mw(context.Background(), inputs, handler(out))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quality gate")
	assert.Contains(t, err.Error(), "input")
}

func TestConsumesTemplate_FailDoesNotCallHandler(t *testing.T) {
	policy := &QualityGatePolicy{
		ConsumesTemplate: "## Required",
	}
	mw := QualityGateMiddleware(policy)

	handlerCalled := false
	h := func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		handlerCalled = true
		return map[string]any{}, nil
	}

	_, err := mw(context.Background(), map[string]any{"x": "no heading here"}, h)
	require.Error(t, err)
	assert.False(t, handlerCalled, "handler must not be called when input pre-check fails")
}

func TestBothTemplates_Pass(t *testing.T) {
	policy := &QualityGatePolicy{
		ConsumesTemplate: "## Input",
		ProducesTemplate: "## Output",
	}
	mw := QualityGateMiddleware(policy)

	inputs := map[string]any{"in": "## Input\nsome data"}
	out := map[string]any{"out": "## Output\nsome result"}
	got, err := mw(context.Background(), inputs, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

func TestProducesTemplate_AsteriskBulletInOutput_Pass(t *testing.T) {
	// Template uses dash-bullet; LLM responds with asterisk-bullet — must pass.
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Bugs\n- **location**: description",
	}
	mw := QualityGateMiddleware(policy)

	out := map[string]any{
		"bugs": "## Bugs\n* **src/main.go:12**: off-by-one error in loop boundary",
	}
	got, err := mw(context.Background(), nil, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

func TestExtractMarkers_AsteriskBoldItem_EmitsNormalizedMarker(t *testing.T) {
	// Template uses * ** style; extractor emits "- **" (normalized).
	// Validator must accept "* **" in output as equivalent.
	markers := extractMarkers("* **location**: description")
	assert.Equal(t, []string{"- **"}, markers)
}

// ─── Handler error passthrough ─────────────────────────────────────────────

func TestHandlerError_Passthrough(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Expected",
	}
	mw := QualityGateMiddleware(policy)

	innerErr := errors.New("LLM provider timeout")
	_, err := mw(context.Background(), nil, failHandler(innerErr))
	require.Error(t, err)
	assert.Equal(t, innerErr, err)
}

// ─── Non-string outputs ────────────────────────────────────────────────────

func TestNonStringOutputsIgnored(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Found",
	}
	mw := QualityGateMiddleware(policy)

	out := map[string]any{
		"count":   42,
		"active":  true,
		"tags":    []any{"a", "b"},
		"message": "## Found\nsome issue",
	}
	got, err := mw(context.Background(), nil, handler(out))
	require.NoError(t, err)
	assert.Equal(t, out, got)
}

func TestNonStringOutputsOnly_MissingStructure(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Expected",
	}
	mw := QualityGateMiddleware(policy)

	out := map[string]any{
		"count":  42,
		"active": true,
	}
	_, err := mw(context.Background(), nil, handler(out))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quality gate")
}

func TestEmptyStringOutput_MissingStructure(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Expected",
	}
	mw := QualityGateMiddleware(policy)

	out := map[string]any{"result": ""}
	_, err := mw(context.Background(), nil, handler(out))
	require.Error(t, err)
}

func TestEmptyOutputMap_MissingStructure(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Expected",
	}
	mw := QualityGateMiddleware(policy)

	_, err := mw(context.Background(), nil, handler(map[string]any{}))
	require.Error(t, err)
}

// ─── DAG integration tests ─────────────────────────────────────────────────

func TestDAGRetryIntegration(t *testing.T) {
	calls := 0
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Findings\n- **location**: description",
	}

	node := &dag.Node{
		ID:       "analyze",
		Kind:     dag.KindTask,
		Consumes: []string{"input"},
		Produces: []string{"output"},
		Run: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			calls++
			if calls == 1 {
				// First attempt: missing required structure
				return map[string]any{
					"output": "I found some issues in the code.",
				}, nil
			}
			// Second attempt: proper structured output
			return map[string]any{
				"output": "## Findings\n- **src/main.go:12**: buffer overflow detected",
			}, nil
		},
		MaxRetries:  1,
		Middlewares: []dag.Middleware{QualityGateMiddleware(policy)},
	}

	d, err := dag.New(node)
	require.NoError(t, err)

	out, err := d.Execute(context.Background(), dag.WithInputs(map[string]any{
		"input": "check this code",
	}))
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "expected exactly 2 calls (1 failed + 1 retry)")
	assert.Contains(t, out["output"].(string), "## Findings")
}

func TestDAGRetryExhausted(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Required",
	}

	node := &dag.Node{
		ID:       "stuck",
		Kind:     dag.KindTask,
		Consumes: []string{"input"},
		Produces: []string{"output"},
		Run: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
			return map[string]any{"output": "some generic response without structure"}, nil
		},
		MaxRetries:  2,
		Middlewares: []dag.Middleware{QualityGateMiddleware(policy)},
	}

	d, err := dag.New(node)
	require.NoError(t, err)

	_, err = d.Execute(context.Background(), dag.WithInputs(map[string]any{
		"input": "check this code",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quality gate")
}

// ─── collectStrings tests ──────────────────────────────────────────────────

func TestCollectStrings(t *testing.T) {
	tests := []struct {
		name    string
		outputs map[string]any
		want    []string
	}{
		{
			name:    "empty map",
			outputs: map[string]any{},
			want:    []string{},
		},
		{
			name:    "single string",
			outputs: map[string]any{"a": "hello"},
			want:    []string{"hello"},
		},
		{
			name:    "mixed types",
			outputs: map[string]any{"a": "hello", "b": 42, "c": "world"},
			want:    []string{"hello", "world"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectStrings(tt.outputs)
			for _, want := range tt.want {
				assert.Contains(t, got, want)
			}
		})
	}
}

// ─── Middleware chain composition ──────────────────────────────────────────

func TestMiddlewareChainComposition(t *testing.T) {
	var order []string

	logging := func(ctx context.Context, inputs map[string]any, next dag.MiddlewareFunc) (map[string]any, error) {
		order = append(order, "logging-pre")
		out, err := next(ctx, inputs)
		order = append(order, "logging-post")
		return out, err
	}

	policy := &QualityGatePolicy{ProducesTemplate: "## OK"}
	gate := QualityGateMiddleware(policy)

	chain := dag.ChainMiddlewares(logging, gate)

	h := handler(map[string]any{"result": "## OK\neverything is fine"})
	out, err := chain(context.Background(), nil, h)
	require.NoError(t, err)
	assert.Contains(t, out["result"].(string), "## OK")
	assert.Equal(t, []string{"logging-pre", "logging-post"}, order)
}

func TestMiddlewareChainComposition_GateRejects(t *testing.T) {
	var outerSawError bool

	logging := func(ctx context.Context, inputs map[string]any, next dag.MiddlewareFunc) (map[string]any, error) {
		out, err := next(ctx, inputs)
		if err != nil {
			outerSawError = true
		}
		return out, err
	}

	policy := &QualityGatePolicy{ProducesTemplate: "## Missing"}
	gate := QualityGateMiddleware(policy)

	chain := dag.ChainMiddlewares(logging, gate)

	h := handler(map[string]any{"result": "no structure here"})
	_, err := chain(context.Background(), nil, h)
	require.Error(t, err)
	assert.True(t, outerSawError)
	assert.Contains(t, err.Error(), "quality gate")
}

// ─── Input/context passthrough ─────────────────────────────────────────────

func TestInputsPassedThrough(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Result",
	}
	mw := QualityGateMiddleware(policy)

	var receivedInputs map[string]any
	h := func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		receivedInputs = inputs
		return map[string]any{"result": "## Result\nok"}, nil
	}

	inputs := map[string]any{"task": "analyze", "code": "func main() {}"}
	_, err := mw(context.Background(), inputs, h)
	require.NoError(t, err)
	assert.Equal(t, inputs, receivedInputs)
}

func TestContextPropagated(t *testing.T) {
	type ctxKey string
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Result",
	}
	mw := QualityGateMiddleware(policy)

	ctx := context.WithValue(context.Background(), ctxKey("test"), "value")
	var receivedCtx context.Context
	h := func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
		receivedCtx = ctx
		return map[string]any{"result": "## Result\nok"}, nil
	}

	_, err := mw(ctx, nil, h)
	require.NoError(t, err)
	assert.Equal(t, "value", receivedCtx.Value(ctxKey("test")))
}

// ─── Concurrent execution ──────────────────────────────────────────────────

func TestConcurrentDAGExecution(t *testing.T) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Done",
	}

	nodes := make([]*dag.Node, 3)
	for i := 0; i < 3; i++ {
		idx := i
		nodes[i] = &dag.Node{
			ID:       fmt.Sprintf("node-%d", idx),
			Kind:     dag.KindTask,
			Produces: []string{fmt.Sprintf("out-%d", idx)},
			Run: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				time.Sleep(time.Millisecond)
				return map[string]any{
					fmt.Sprintf("out-%d", idx): fmt.Sprintf("## Done\ntask %d complete", idx),
				}, nil
			},
			Middlewares: []dag.Middleware{QualityGateMiddleware(policy)},
		}
	}

	d, err := dag.New(nodes...)
	require.NoError(t, err)

	out, err := d.Execute(context.Background())
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("out-%d", i)
		assert.Contains(t, out[key].(string), "## Done")
	}
}

// ─── Benchmarks ────────────────────────────────────────────────────────────

func BenchmarkQualityGateMiddleware_NoTemplates(b *testing.B) {
	mw := QualityGateMiddleware(nil)
	out := map[string]any{"result": "hello world"}
	h := handler(out)
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mw(ctx, nil, h)
	}
}

func BenchmarkQualityGateMiddleware_Template_Pass(b *testing.B) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Findings\n- **location**: description",
	}
	mw := QualityGateMiddleware(policy)
	out := map[string]any{
		"result": strings.Repeat("## Findings\n- **src/main.go**: issue found. ", 10),
	}
	h := handler(out)
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mw(ctx, nil, h)
	}
}

func BenchmarkQualityGateMiddleware_Template_Fail(b *testing.B) {
	policy := &QualityGatePolicy{
		ProducesTemplate: "## Findings\n- **location**: description",
	}
	mw := QualityGateMiddleware(policy)
	out := map[string]any{
		"result": strings.Repeat("I found some issues in the code. ", 10),
	}
	h := handler(out)
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mw(ctx, nil, h)
	}
}
