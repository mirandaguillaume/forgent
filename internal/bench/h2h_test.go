package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadContestants(t *testing.T) {
	// Create temp dir with test fixtures
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-a.md"), []byte("You are agent A. Review code."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-b.md"), []byte("You are agent B. Analyze code."), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "not-md.txt"), []byte("ignored"), 0644))

	contestants, err := LoadContestants(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, len(contestants), "should load 2 .md files")
	assert.Equal(t, "agent-a", contestants[0].Name)
	assert.Equal(t, "agent-b", contestants[1].Name)
	assert.Greater(t, contestants[0].Words, 0)
}

func TestLoadContestants_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	contestants, err := LoadContestants(dir)
	require.NoError(t, err)
	assert.Empty(t, contestants)
}

func TestLoadContestants_MissingDir(t *testing.T) {
	_, err := LoadContestants("/nonexistent/dir")
	require.Error(t, err)
}

func TestLoadForgentContestants(t *testing.T) {
	// Create temp imported dir structure
	dir := t.TempDir()
	stdDir := filepath.Join(dir, "test-source", "output-standard")
	compactDir := filepath.Join(dir, "test-source", "output-compact")
	require.NoError(t, os.MkdirAll(stdDir, 0755))
	require.NoError(t, os.MkdirAll(compactDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(stdDir, "agent.md"), []byte("standard output here"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(compactDir, "agent.md"), []byte("compact output"), 0644))

	contestants, err := LoadForgentContestants(dir)
	require.NoError(t, err)
	assert.Equal(t, 2, len(contestants))
	assert.Equal(t, "test-source/forgent-standard", contestants[0].Name)
	assert.Equal(t, "test-source/forgent-compact", contestants[1].Name)
	assert.Greater(t, contestants[0].Words, contestants[1].Words)
}

func TestParseH2HJudgeResponse_Valid(t *testing.T) {
	raw := `{"criteria_hits": [true, false, true], "reasoning": "good review"}`
	result, err := parseH2HJudgeResponse(raw, 3)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, true}, result.CriteriaHits)
	assert.Equal(t, "good review", result.Reasoning)
}

func TestParseH2HJudgeResponse_WrappedInMarkdown(t *testing.T) {
	raw := "```json\n{\"criteria_hits\": [true], \"reasoning\": \"ok\"}\n```"
	result, err := parseH2HJudgeResponse(raw, 1)
	require.NoError(t, err)
	assert.Equal(t, []bool{true}, result.CriteriaHits)
}

func TestParseH2HJudgeResponse_PadsCriteria(t *testing.T) {
	raw := `{"criteria_hits": [true], "reasoning": "partial"}`
	result, err := parseH2HJudgeResponse(raw, 3)
	require.NoError(t, err)
	assert.Equal(t, 3, len(result.CriteriaHits), "should pad to expected count")
	assert.True(t, result.CriteriaHits[0])
	assert.False(t, result.CriteriaHits[1])
	assert.False(t, result.CriteriaHits[2])
}

func TestParseH2HJudgeResponse_TruncatesCriteria(t *testing.T) {
	raw := `{"criteria_hits": [true, true, true, true], "reasoning": "extra"}`
	result, err := parseH2HJudgeResponse(raw, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(result.CriteriaHits), "should truncate to expected count")
}

func TestParseH2HJudgeResponse_AlternativeReasoning(t *testing.T) {
	raw := `{"criteria_hits": [true], "explanation": "good"}`
	result, err := parseH2HJudgeResponse(raw, 1)
	require.NoError(t, err)
	assert.Equal(t, "good", result.Reasoning)
}

func TestSeverityWeight(t *testing.T) {
	assert.Equal(t, 4.0, SeverityWeight("Critical"))
	assert.Equal(t, 3.0, SeverityWeight("High"))
	assert.Equal(t, 2.0, SeverityWeight("Medium"))
	assert.Equal(t, 1.0, SeverityWeight("Low"))
	assert.Equal(t, 1.0, SeverityWeight("unknown"))
	assert.Equal(t, 4.0, SeverityWeight("critical")) // case-insensitive
}

func TestParseH2HJudgeResponse_InvalidJSON(t *testing.T) {
	_, err := parseH2HJudgeResponse("not json", 1)
	require.Error(t, err)
}

func TestEstimateCost(t *testing.T) {
	// Zero words → zero cost
	assert.Equal(t, 0.0, EstimateCost(0, 0))

	// 1M words input only → ~$4 (1M * 1.33 tokens * $3/MTok)
	cost := EstimateCost(1_000_000, 0)
	assert.InDelta(t, 3.99, cost, 0.01)

	// 1M words output only → ~$20 (1M * 1.33 tokens * $15/MTok)
	cost = EstimateCost(0, 1_000_000)
	assert.InDelta(t, 19.95, cost, 0.01)

	// Mixed: small realistic bench (50k input, 10k output)
	cost = EstimateCost(50_000, 10_000)
	assert.Greater(t, cost, 0.0)
	assert.Less(t, cost, 1.0) // should be well under $1
}

func TestH2HTokens(t *testing.T) {
	dir := t.TempDir()
	fixturesDir := filepath.Join(dir, "contestants")
	importedDir := filepath.Join(dir, "imported")

	require.NoError(t, os.MkdirAll(fixturesDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(fixturesDir, "src-agent.md"),
		[]byte("You are a code reviewer. Check for bugs, security issues, and best practices. Be thorough."), 0644))

	stdDir := filepath.Join(importedDir, "src", "output-standard")
	compDir := filepath.Join(importedDir, "src", "output-compact")
	require.NoError(t, os.MkdirAll(stdDir, 0755))
	require.NoError(t, os.MkdirAll(compDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(stdDir, "agent.md"), []byte("Standard output with more words for structure"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(compDir, "agent.md"), []byte("Compact output terse"), 0644))

	result, err := RunH2HTokens(fixturesDir, importedDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Entries), 3, "should have hand-written + standard + compact")
	assert.GreaterOrEqual(t, len(result.Sources), 1, "should have at least 1 source comparison")

	for _, src := range result.Sources {
		assert.Greater(t, src.HandWritten, 0, "should have hand-written words for %s", src.Source)
		assert.Greater(t, src.Standard, 0, "should have standard words for %s", src.Source)
		assert.Greater(t, src.Compact, 0, "should have compact words for %s", src.Source)
	}
}

func TestH2HTokens_RealFixtures(t *testing.T) {
	fixturesDir := "fixtures/contestants"
	importedDir := "fixtures/imported"
	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Skip("fixtures not found")
	}

	result, err := RunH2HTokens(fixturesDir, importedDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Sources), 4, "should have 4 source comparisons")

	totalHW, totalStd, totalCompact := 0, 0, 0
	for _, src := range result.Sources {
		t.Logf("  %-12s  hand-written=%4d  standard=%4d (%.0f%% less)  compact=%4d (%.0f%% less)",
			src.Source, src.HandWritten, src.Standard, src.StandardSaving, src.Compact, src.CompactSaving)
		assert.Greater(t, src.StandardSaving, 0.0, "standard should be lighter than hand-written for %s", src.Source)
		assert.Greater(t, src.CompactSaving, src.StandardSaving, "compact should save more than standard for %s", src.Source)
		totalHW += src.HandWritten
		totalStd += src.Standard
		totalCompact += src.Compact
	}

	avgStdSaving := (1.0 - float64(totalStd)/float64(totalHW)) * 100
	avgCompactSaving := (1.0 - float64(totalCompact)/float64(totalHW)) * 100
	t.Logf("\n  TOTAL         hand-written=%4d  standard=%4d (%.0f%% less)  compact=%4d (%.0f%% less)",
		totalHW, totalStd, avgStdSaving, totalCompact, avgCompactSaving)
}

func TestLoadContestants_RealFixtures(t *testing.T) {
	// Test with actual fixture files if they exist
	fixturesDir := "fixtures/contestants"
	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Skip("fixtures/contestants not found")
	}
	contestants, err := LoadContestants(fixturesDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(contestants), 4, "should have at least 4 hand-written agents")
	for _, c := range contestants {
		assert.Greater(t, c.Words, 0, "contestant %s should have words", c.Name)
		t.Logf("  %s: %d words", c.Name, c.Words)
	}
}

func TestMartianTasks_Integration(t *testing.T) {
	martianDir := "testdata/martian"
	if _, err := os.Stat(martianDir); os.IsNotExist(err) {
		t.Skip("testdata/martian not found")
	}

	tasks, err := MartianTasks(martianDir)
	require.NoError(t, err)

	categories := make(map[string]int)
	totalCriteria := 0
	for _, task := range tasks {
		categories[task.Category]++
		totalCriteria += len(task.Criteria)
	}
	t.Logf("Loaded %d tasks with %d criteria across %d categories", len(tasks), totalCriteria, len(categories))
	for cat, count := range categories {
		t.Logf("  %s: %d PRs", cat, count)
	}

	for _, task := range tasks {
		assert.NotEmpty(t, task.URL, "martian task %s should have URL", task.ID)
		assert.Empty(t, task.Code, "martian task %s should not have inline code", task.ID)
	}
}

func TestLoadForgentContestants_RealFixtures(t *testing.T) {
	importedDir := "fixtures/imported"
	if _, err := os.Stat(importedDir); os.IsNotExist(err) {
		t.Skip("fixtures/imported not found")
	}
	contestants, err := LoadForgentContestants(importedDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(contestants), 4, "should have at least 4 forgent variants (2 per source)")
	for _, c := range contestants {
		assert.Greater(t, c.Words, 0, "contestant %s should have words", c.Name)
		t.Logf("  %s: %d words", c.Name, c.Words)
	}
}
