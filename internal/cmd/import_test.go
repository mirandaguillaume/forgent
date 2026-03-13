package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		provider string
		expected string
		wantErr  bool
	}{
		{
			name:     "FORGENT_API_KEY takes priority",
			env:      map[string]string{"FORGENT_API_KEY": "forgent-key"},
			provider: "anthropic",
			expected: "forgent-key",
		},
		{
			name:     "falls back to ANTHROPIC_API_KEY",
			env:      map[string]string{"ANTHROPIC_API_KEY": "anthropic-key"},
			provider: "anthropic",
			expected: "anthropic-key",
		},
		{
			name:     "falls back to OPENAI_API_KEY",
			env:      map[string]string{"OPENAI_API_KEY": "openai-key"},
			provider: "openai",
			expected: "openai-key",
		},
		{
			name:     "no key found",
			env:      map[string]string{},
			provider: "anthropic",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := resolveAPIKey(tt.provider, tt.env)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, key)
			}
		})
	}
}
