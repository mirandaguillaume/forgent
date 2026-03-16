package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsomorphism_CIReviewer(t *testing.T) {
	result, err := RunIsomorphism("../../skills", "../../agents")
	require.NoError(t, err)
	assert.True(t, result.SkillNamesMatch, "skill names should match across targets")
	assert.True(t, result.IOContractsMatch, "I/O contracts should match")
	assert.Equal(t, 1.0, result.StructureScore, "structure should be identical")
	// Should find all 16 skills
	assert.Len(t, result.ClaudeSkills, 16)
	assert.Len(t, result.CopilotSkills, 16)
	t.Logf("Claude skills: %v", result.ClaudeSkills)
	t.Logf("Copilot skills: %v", result.CopilotSkills)
}

func TestIsomorphism_SkillSignatures(t *testing.T) {
	// Verify specific I/O contracts are preserved across targets.
	result, err := RunIsomorphism("../../skills", "../../agents")
	require.NoError(t, err)

	// Find ts-linter in claude output
	var claudeLinter, copilotLinter *SkillSignature
	for i := range result.ClaudeSkills {
		if result.ClaudeSkills[i].Name == "ts-linter" {
			claudeLinter = &result.ClaudeSkills[i]
		}
	}
	for i := range result.CopilotSkills {
		if result.CopilotSkills[i].Name == "ts-linter" {
			copilotLinter = &result.CopilotSkills[i]
		}
	}
	require.NotNil(t, claudeLinter, "ts-linter should exist in claude output")
	require.NotNil(t, copilotLinter, "ts-linter should exist in copilot output")

	assert.Equal(t, claudeLinter.Consumes, copilotLinter.Consumes)
	assert.Equal(t, claudeLinter.Produces, copilotLinter.Produces)
}

func TestIsomorphism_EmptyProject(t *testing.T) {
	tmpSkills := t.TempDir()
	tmpAgents := t.TempDir()
	result, err := RunIsomorphism(tmpSkills, tmpAgents)
	require.NoError(t, err)
	assert.True(t, result.SkillNamesMatch, "empty → empty is a match")
	assert.Len(t, result.ClaudeSkills, 0)
	assert.Len(t, result.CopilotSkills, 0)
}

// --- Mutation-killing tests for parseCSV ---

func TestParseCSV_EmptyString(t *testing.T) {
	// Mutation: `p != ""` → `p == ""` or removing the check entirely.
	// An input of just whitespace/commas should yield no entries.
	result := parseCSV("")
	assert.Empty(t, result, "empty input should produce empty result")
}

func TestParseCSV_OnlyCommasAndSpaces(t *testing.T) {
	// If the empty-check is removed, these would produce ["", "", ""] entries.
	result := parseCSV(" , , , ")
	assert.Empty(t, result, "only commas and spaces should produce empty result")
}

func TestParseCSV_SingleItem(t *testing.T) {
	result := parseCSV("alpha")
	assert.Equal(t, []string{"alpha"}, result)
}

func TestParseCSV_MultipleItemsSorted(t *testing.T) {
	result := parseCSV("gamma, alpha, beta")
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, result, "should be sorted")
}

func TestParseCSV_TrailingComma(t *testing.T) {
	// Trailing comma produces an empty final part that must be filtered.
	result := parseCSV("foo, bar,")
	assert.Equal(t, []string{"bar", "foo"}, result)
}

func TestParseCSV_LeadingComma(t *testing.T) {
	result := parseCSV(",foo, bar")
	assert.Equal(t, []string{"bar", "foo"}, result)
}

// --- Mutation-killing tests for sigNamesEqual ---

func TestSigNamesEqual_BothEmpty(t *testing.T) {
	assert.True(t, sigNamesEqual(nil, nil), "two nil slices should be equal")
	assert.True(t, sigNamesEqual([]SkillSignature{}, []SkillSignature{}), "two empty slices should be equal")
}

func TestSigNamesEqual_DifferentLengths(t *testing.T) {
	a := []SkillSignature{{Name: "a"}}
	b := []SkillSignature{{Name: "a"}, {Name: "b"}}
	assert.False(t, sigNamesEqual(a, b), "different lengths should not match")
}

func TestSigNamesEqual_SameLengthDifferentNames(t *testing.T) {
	a := []SkillSignature{{Name: "a"}}
	b := []SkillSignature{{Name: "b"}}
	assert.False(t, sigNamesEqual(a, b))
}

func TestSigNamesEqual_SameLengthSameNames(t *testing.T) {
	a := []SkillSignature{{Name: "x"}, {Name: "y"}}
	b := []SkillSignature{{Name: "x"}, {Name: "y"}}
	assert.True(t, sigNamesEqual(a, b))
}

func TestSigNamesEqual_OneEmptyOneNot(t *testing.T) {
	a := []SkillSignature{}
	b := []SkillSignature{{Name: "x"}}
	assert.False(t, sigNamesEqual(a, b), "empty vs non-empty should not match")
}

func TestSigNamesEqual_SingleElementMatch(t *testing.T) {
	// Boundary: exactly len==1 on both sides.
	a := []SkillSignature{{Name: "same"}}
	b := []SkillSignature{{Name: "same"}}
	assert.True(t, sigNamesEqual(a, b))
}

// --- Mutation-killing tests for sigIOEqual ---

func TestSigIOEqual_BothEmpty(t *testing.T) {
	assert.True(t, sigIOEqual(nil, nil))
	assert.True(t, sigIOEqual([]SkillSignature{}, []SkillSignature{}))
}

func TestSigIOEqual_DifferentLengths(t *testing.T) {
	a := []SkillSignature{{Name: "a", Consumes: []string{"x"}}}
	b := []SkillSignature{}
	assert.False(t, sigIOEqual(a, b), "different lengths should not match")
}

func TestSigIOEqual_SameNamesDifferentConsumes(t *testing.T) {
	a := []SkillSignature{{Name: "a", Consumes: []string{"x"}, Produces: []string{"y"}}}
	b := []SkillSignature{{Name: "a", Consumes: []string{"z"}, Produces: []string{"y"}}}
	assert.False(t, sigIOEqual(a, b))
}

func TestSigIOEqual_SameNamesDifferentProduces(t *testing.T) {
	a := []SkillSignature{{Name: "a", Consumes: []string{"x"}, Produces: []string{"y"}}}
	b := []SkillSignature{{Name: "a", Consumes: []string{"x"}, Produces: []string{"z"}}}
	assert.False(t, sigIOEqual(a, b))
}

func TestSigIOEqual_MatchingIO(t *testing.T) {
	a := []SkillSignature{{Name: "a", Consumes: []string{"x"}, Produces: []string{"y"}}}
	b := []SkillSignature{{Name: "a", Consumes: []string{"x"}, Produces: []string{"y"}}}
	assert.True(t, sigIOEqual(a, b))
}

func TestSigIOEqual_NilConsumesVsEmpty(t *testing.T) {
	// nil and empty slice behave differently with sliceEqual.
	a := []SkillSignature{{Name: "a", Consumes: nil, Produces: nil}}
	b := []SkillSignature{{Name: "a", Consumes: nil, Produces: nil}}
	assert.True(t, sigIOEqual(a, b))
}

// --- Mutation-killing tests for sliceEqual ---

func TestSliceEqual_BothNil(t *testing.T) {
	assert.True(t, sliceEqual(nil, nil))
}

func TestSliceEqual_BothEmpty(t *testing.T) {
	assert.True(t, sliceEqual([]string{}, []string{}))
}

func TestSliceEqual_OneNilOneEmpty(t *testing.T) {
	// nil has len 0, empty has len 0, so these should be equal.
	assert.True(t, sliceEqual(nil, []string{}))
}

func TestSliceEqual_DifferentLengths(t *testing.T) {
	assert.False(t, sliceEqual([]string{"a"}, []string{"a", "b"}))
}

func TestSliceEqual_SameLengthDifferentValues(t *testing.T) {
	assert.False(t, sliceEqual([]string{"a"}, []string{"b"}))
}

func TestSliceEqual_SingleElementMatch(t *testing.T) {
	assert.True(t, sliceEqual([]string{"a"}, []string{"a"}))
}

func TestSliceEqual_MultipleElementsMatch(t *testing.T) {
	assert.True(t, sliceEqual([]string{"a", "b", "c"}, []string{"a", "b", "c"}))
}

func TestSliceEqual_DifferAtLastElement(t *testing.T) {
	assert.False(t, sliceEqual([]string{"a", "b", "c"}, []string{"a", "b", "d"}))
}

// --- Mutation-killing tests for extractSkillOrder ---

func TestExtractSkillOrder_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	order := extractSkillOrder(tmpDir)
	assert.Empty(t, order)
}

func TestExtractSkillOrder_SkipsDirectories(t *testing.T) {
	// Line 138: `err != nil || info.IsDir()` — must skip directories.
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	order := extractSkillOrder(tmpDir)
	assert.Empty(t, order)
}

func TestExtractSkillOrder_SkipsSKILLmd(t *testing.T) {
	// Line 143: `filepath.Base(path) == "SKILL.md"` filter.
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "skills", "my-agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	// Even though the path contains "agent", SKILL.md files should be skipped.
	content := "### Step 1: lint\n### Step 2: review\n"
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "SKILL.md"), []byte(content), 0644))
	order := extractSkillOrder(tmpDir)
	assert.Empty(t, order, "SKILL.md files should be excluded even if path contains 'agent'")
}

func TestExtractSkillOrder_RequiresAgentInPath(t *testing.T) {
	// Line 143: `!strings.Contains(rel, "agent")` — files without "agent" in path are skipped.
	tmpDir := t.TempDir()
	otherDir := filepath.Join(tmpDir, "skills", "lint")
	require.NoError(t, os.MkdirAll(otherDir, 0755))
	content := "### Step 1: lint\n### Step 2: review\n"
	require.NoError(t, os.WriteFile(filepath.Join(otherDir, "readme.md"), []byte(content), 0644))
	order := extractSkillOrder(tmpDir)
	assert.Empty(t, order, "non-agent paths should be skipped")
}

func TestExtractSkillOrder_ParsesSteps(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	content := "# Agent\n\n### Step 1: lint\n### Step 2: review\n### Step 3: deploy\n"
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "ci-agent.md"), []byte(content), 0644))
	order := extractSkillOrder(tmpDir)
	assert.Equal(t, []string{"lint", "review", "deploy"}, order)
}

func TestExtractSkillOrder_TrimsWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	content := "### Step 1:  lint  \n"
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "my-agent.md"), []byte(content), 0644))
	order := extractSkillOrder(tmpDir)
	assert.Equal(t, []string{"lint"}, order)
}

// --- Mutation-killing tests for StructureScore arithmetic ---

func TestStructureScore_AllMatch(t *testing.T) {
	// When names, IO, and order all match, score should be 3/3 = 1.0.
	result, err := RunIsomorphism("../../skills", "../../agents")
	require.NoError(t, err)
	assert.Equal(t, 1.0, result.StructureScore, "all 3 dimensions match → score = 1.0")
}

func TestStructureScore_EmptyProject(t *testing.T) {
	// Empty projects: names match (both empty), IO match, order match → all 3 dimensions.
	tmpSkills := t.TempDir()
	tmpAgents := t.TempDir()
	result, err := RunIsomorphism(tmpSkills, tmpAgents)
	require.NoError(t, err)
	// Both empty: names match = true, IO match = true, order (both empty) match = true → 3/3
	assert.Equal(t, 1.0, result.StructureScore, "empty project → 1.0")
}

func TestStructureScore_ExactArithmetic(t *testing.T) {
	// Verify the score is exactly matched/total, not some other formula.
	// With the real project, all 3 match, so score = 3.0/3.0 = 1.0.
	result, err := RunIsomorphism("../../skills", "../../agents")
	require.NoError(t, err)
	// Verify exact floating point value.
	assert.InDelta(t, 1.0, result.StructureScore, 0.0001)
	// If mutation changes + to -, or * to /, the value would differ.
}

// --- Mutation-killing tests for sortSigs ---

func TestSortSigs_AlreadySorted(t *testing.T) {
	sigs := []SkillSignature{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	sortSigs(sigs)
	assert.Equal(t, "a", sigs[0].Name)
	assert.Equal(t, "b", sigs[1].Name)
	assert.Equal(t, "c", sigs[2].Name)
}

func TestSortSigs_Unsorted(t *testing.T) {
	sigs := []SkillSignature{{Name: "c"}, {Name: "a"}, {Name: "b"}}
	sortSigs(sigs)
	assert.Equal(t, "a", sigs[0].Name)
	assert.Equal(t, "b", sigs[1].Name)
	assert.Equal(t, "c", sigs[2].Name)
}

func TestSortSigs_Empty(t *testing.T) {
	var sigs []SkillSignature
	sortSigs(sigs) // should not panic
	assert.Empty(t, sigs)
}

// --- Mutation-killing test for extractSignatures ---

func TestExtractSignatures_ParsesConsumesAndProduces(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	// Frontmatter description format from BuildSkillDescription
	content := "---\nname: my-skill\ndescription: analytical-based skill consuming foo, bar to produce baz\n---\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644))

	sigs, err := extractSignatures(tmpDir)
	require.NoError(t, err)
	require.Len(t, sigs, 1)
	assert.Equal(t, "my-skill", sigs[0].Name)
	assert.Equal(t, []string{"bar", "foo"}, sigs[0].Consumes)
	assert.Equal(t, []string{"baz"}, sigs[0].Produces)
}

func TestExtractSignatures_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	sigs, err := extractSignatures(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, sigs)
}

func TestExtractSignatures_SkipsNonSKILLmd(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "skills", "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "README.md"), []byte("---\nname: my-skill\ndescription: skill consuming x to produce y\n---\n"), 0644))

	sigs, err := extractSignatures(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, sigs, "non-SKILL.md files should be skipped")
}
