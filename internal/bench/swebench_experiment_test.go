package bench

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/bench/experiment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractUnifiedDiffFromOutput verifies patch extraction from combined outputs.
func TestExtractUnifiedDiffFromOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHas  string // non-empty: expected substring
		wantNone bool   // true: expect empty result
	}{
		{
			name:     "no diff in output",
			input:    "I analyzed the issue. The problem is in the handler function.",
			wantNone: true,
		},
		{
			name: "clean unified diff",
			input: `Here is the fix:

--- a/src/handler.go
+++ b/src/handler.go
@@ -10,6 +10,7 @@ func handleRequest(r *http.Request) {
 	ctx := r.Context()
-	return process(ctx)
+	return process(ctx, timeout)
 }`,
			wantHas: "--- a/src/handler.go",
		},
		{
			name: "diff with trailing explanation",
			input: `--- a/lib/auth.py
+++ b/lib/auth.py
@@ -5,4 +5,5 @@
 def validate_token(token):
-    return token is not None
+    if token is None:
+        raise ValueError("token required")
+    return True

This fix ensures that None tokens raise an error instead of silently passing.`,
			wantHas: "+++ b/lib/auth.py",
		},
		{
			name: "multi-file diff",
			input: `--- a/pkg/a.go
+++ b/pkg/a.go
@@ -1,3 +1,3 @@
-func A() {}
+func A() { return }
--- a/pkg/b.go
+++ b/pkg/b.go
@@ -1,3 +1,3 @@
-func B() {}
+func B() { return }`,
			wantHas: "--- a/pkg/b.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractUnifiedDiffFromOutput(tc.input)
			if tc.wantNone {
				assert.Empty(t, got)
			} else {
				assert.Contains(t, got, tc.wantHas)
			}
		})
	}
}

// TestBuildSWETaskDef verifies that the TaskDefinition is built from a SWE-bench task.
func TestBuildSWETaskDef(t *testing.T) {
	tmpl := sampleSWETemplate()
	sweTask := SWEBenchTask{
		InstanceID:       "django__django-11099",
		Repo:             "django/django",
		ProblemStatement: "Fix the bug in QuerySet.order_by().",
		GoldPatch: `--- a/django/db/models/query.py
+++ b/django/db/models/query.py
@@ -1,3 +1,3 @@
-    return self._clone()
+    return self._clone().order_by(*field_names)`,
		HintsText: "Look at the order_by method.",
	}

	td := buildSWETaskDef(tmpl, sweTask)

	assert.Equal(t, "django__django-11099", td.ID)
	assert.Contains(t, td.Input, "Fix the bug in QuerySet.order_by()")
	assert.Contains(t, td.Input, "django/django")
	assert.NotContains(t, td.Input, "django/db/models/query.py", "gold patch file list must not leak into prompt")
	assert.Contains(t, td.Input, "Look at the order_by method")
}

// TestSWEBenchExperimentResult_Rates verifies metric computation.
func TestSWEBenchExperimentResult_Rates(t *testing.T) {
	result := &SWEBenchExperimentResult{
		Tasks: 4,
		Results: map[string][]SWETaskResult{
			"text": {
				{InstanceID: "t1", Patch: "diff", Applied: true},
				{InstanceID: "t2"},
				{InstanceID: "t3", Patch: "diff", Applied: false},
				{InstanceID: "t4", Patch: "diff", Applied: true, TestPassed: true},
			},
		},
	}

	assert.InDelta(t, 75.0, result.PatchRate("text"), 0.01)   // 3/4 have patch
	assert.InDelta(t, 50.0, result.ApplyRate("text"), 0.01)   // 2/4 applied
	assert.InDelta(t, 25.0, result.ResolveRate("text"), 0.01) // 1/4 test passed
	assert.Equal(t, 0.0, result.PatchRate("runtime"))          // unknown condition → 0
}

// TestBuildSWECodeSummary verifies code context extraction from gold patch.
func TestBuildSWECodeSummary(t *testing.T) {
	task := SWEBenchTask{
		InstanceID:       "flask__flask-1234",
		Repo:             "pallets/flask",
		ProblemStatement: "Request context not cleaned up on error.",
		GoldPatch: `--- a/src/flask/ctx.py
+++ b/src/flask/ctx.py
@@ -10,4 +10,5 @@
+    self._cleanup()`,
	}

	summary := buildSWECodeSummary(task)
	assert.Contains(t, summary, "pallets/flask")
	assert.NotContains(t, summary, "src/flask/ctx.py", "gold patch file list must not appear in code summary")
}

// TestBuildSWECodeSummary_EmptyPatch verifies fallback when no gold patch.
func TestBuildSWECodeSummary_EmptyPatch(t *testing.T) {
	task := SWEBenchTask{
		InstanceID: "no-patch",
		Repo:       "owner/repo",
	}
	summary := buildSWECodeSummary(task)
	assert.Contains(t, summary, "owner/repo")
}

// TestRunSWEBenchExperiment_UnsupportedCondition verifies input validation.
func TestRunSWEBenchExperiment_UnsupportedCondition(t *testing.T) {
	_, err := RunSWEBenchExperiment(SWEBenchExperimentConfig{
		Tasks:      []SWEBenchTask{{InstanceID: "t1"}},
		Template:   sampleSWETemplate(),
		Provider:   nil,
		Conditions: []string{"invalid"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported condition")
}

// sampleSWETemplate returns a minimal TaskDefinition for test use.
func sampleSWETemplate() experiment.TaskDefinition {
	return experiment.TaskDefinition{
		ID:   "resolve-issue",
		Name: "Issue Resolver",
		Steps: []experiment.StepDef{
			{
				ID:       "read-issue",
				Name:     "Read Issue",
				Prompt:   "Read the issue.",
				Produces: []experiment.SlotSpec{{Key: "analysis"}},
			},
			{
				ID:       "generate-patch",
				Name:     "Generate Patch",
				Prompt:   "Generate patch.",
				Consumes: []experiment.SlotSpec{{Key: "analysis"}},
				Produces: []experiment.SlotSpec{{Key: "patch"}},
			},
		},
	}
}
