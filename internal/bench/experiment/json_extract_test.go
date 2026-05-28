package experiment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractJSON_BareObject(t *testing.T) {
	m, err := extractJSON(`{"ruling": "approved", "score": 42}`)
	require.NoError(t, err)
	assert.Equal(t, "approved", m["ruling"])
	assert.InDelta(t, 42.0, m["score"], 0.001)
}

func TestExtractJSON_CodeFence_WithLangTag(t *testing.T) {
	m, err := extractJSON("```json\n{\"ruling\": \"rejected\"}\n```")
	require.NoError(t, err)
	assert.Equal(t, "rejected", m["ruling"])
}

func TestExtractJSON_CodeFence_NoLangTag(t *testing.T) {
	m, err := extractJSON("```\n{\"verdict\": \"pass\"}\n```")
	require.NoError(t, err)
	assert.Equal(t, "pass", m["verdict"])
}

func TestExtractJSON_Preamble(t *testing.T) {
	input := "Here is my structured response:\n{\"ruling\": \"approved\", \"rationale\": \"all good\"}"
	m, err := extractJSON(input)
	require.NoError(t, err)
	assert.Equal(t, "approved", m["ruling"])
}

func TestExtractJSON_BracesInStrings(t *testing.T) {
	// Braces inside string values must not confuse the depth counter.
	input := `{"msg": "value with {brace}", "ok": true}`
	m, err := extractJSON(input)
	require.NoError(t, err)
	assert.Equal(t, "value with {brace}", m["msg"])
	assert.Equal(t, true, m["ok"])
}

func TestExtractJSON_Malformed_ReturnsError(t *testing.T) {
	_, err := extractJSON("{not valid json}")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "json extraction:"), "error must have prefix: %v", err)
}

func TestExtractJSON_Empty_ReturnsError(t *testing.T) {
	_, err := extractJSON("")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "json extraction:"), err)
}

func TestExtractJSON_Whitespace_ReturnsError(t *testing.T) {
	_, err := extractJSON("   \n\t  ")
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "json extraction:"), err)
}

func TestExtractJSON_NoBraces_ReturnsError(t *testing.T) {
	_, err := extractJSON("just some plain text with no JSON")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json extraction:")
}

func TestExtractJSON_Array_ReturnsError(t *testing.T) {
	_, err := extractJSON(`["a", "b", "c"]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json extraction:")
}

func TestExtractJSON_WithTrailingText(t *testing.T) {
	// JSON followed by prose — brace scan finds the object, ignores the rest.
	input := `{"verdict": "approved"} and some trailing commentary.`
	m, err := extractJSON(input)
	require.NoError(t, err)
	assert.Equal(t, "approved", m["verdict"])
}
