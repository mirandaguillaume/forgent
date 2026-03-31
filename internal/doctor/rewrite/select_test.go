package rewrite

import (
	"strings"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFindings = []llmutil.Finding{
	{Category: "guardrails", Issue: "No guardrails", Confidence: "high"},
	{Category: "security", Issue: "No security", Confidence: "high"},
	{Category: "testing", Issue: "No tests", Confidence: "moderate"},
}

func TestSelectRecommendations_NonTTY(t *testing.T) {
	selected, err := SelectRecommendations(testFindings, false, nil)
	require.NoError(t, err)
	assert.Equal(t, testFindings, selected)
}

func TestSelectRecommendations_All(t *testing.T) {
	reader := strings.NewReader("all\n")
	selected, err := SelectRecommendations(testFindings, true, reader)
	require.NoError(t, err)
	assert.Equal(t, testFindings, selected)
}

func TestSelectRecommendations_SingleSelection(t *testing.T) {
	reader := strings.NewReader("2\n")
	selected, err := SelectRecommendations(testFindings, true, reader)
	require.NoError(t, err)
	require.Len(t, selected, 1)
	assert.Equal(t, "security", selected[0].Category)
}

func TestSelectRecommendations_MultipleSelection(t *testing.T) {
	reader := strings.NewReader("1,3\n")
	selected, err := SelectRecommendations(testFindings, true, reader)
	require.NoError(t, err)
	require.Len(t, selected, 2)
	assert.Equal(t, "guardrails", selected[0].Category)
	assert.Equal(t, "testing", selected[1].Category)
}

func TestSelectRecommendations_OutOfRange(t *testing.T) {
	reader := strings.NewReader("5\n")
	_, err := SelectRecommendations(testFindings, true, reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestSelectRecommendations_InvalidInput(t *testing.T) {
	reader := strings.NewReader("abc\n")
	_, err := SelectRecommendations(testFindings, true, reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid selection")
}

func TestSelectRecommendations_Empty(t *testing.T) {
	reader := strings.NewReader("\n")
	_, err := SelectRecommendations(testFindings, true, reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty selection")
}

func TestSelectRecommendations_DuplicateSelection(t *testing.T) {
	reader := strings.NewReader("1,1,2\n")
	selected, err := SelectRecommendations(testFindings, true, reader)
	require.NoError(t, err)
	assert.Len(t, selected, 2, "duplicates should be deduplicated")
}

func TestSelectRecommendations_NoFindings(t *testing.T) {
	selected, err := SelectRecommendations(nil, true, nil)
	require.NoError(t, err)
	assert.Nil(t, selected)
}
