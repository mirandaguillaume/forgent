package static

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/stretchr/testify/assert"
)

func TestSecrets_DetectsOpenAIKey(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("Use this key: sk-abcdefghijklmnopqrstuvwx"),
	}
	check := &secretsCheck{}
	findings := check.Run(analysis)
	assert.Len(t, findings, 1)
	assert.Contains(t, findings[0].Issue, "OpenAI")
	assert.Equal(t, "high", findings[0].Confidence)
}

func TestSecrets_DetectsAWSKey(t *testing.T) {
	// Use a realistic-looking key (not containing "example" which triggers placeholder detection)
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("AWS_ACCESS_KEY_ID=AKIAIOSFODNN7ABCDEFG"),
	}
	check := &secretsCheck{}
	findings := check.Run(analysis)
	assert.Len(t, findings, 1)
	assert.Contains(t, findings[0].Issue, "AWS")
}

func TestSecrets_DetectsGitHubPAT(t *testing.T) {
	// ghp_ + exactly 36 alphanumeric chars
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"),
	}
	check := &secretsCheck{}
	findings := check.Run(analysis)
	assert.Len(t, findings, 1)
	assert.Contains(t, findings[0].Issue, "GitHub PAT")
}

func TestSecrets_SkipsPlaceholders(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("Example: sk-your_api_key_here_replace_me_placeholder"),
	}
	check := &secretsCheck{}
	findings := check.Run(analysis)
	assert.Empty(t, findings, "placeholder indicators should suppress finding")
}

func TestSecrets_DetectsPasswordAssignment(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("password = mysecretpassword123"),
	}
	check := &secretsCheck{}
	findings := check.Run(analysis)
	assert.Len(t, findings, 1)
	assert.Contains(t, findings[0].Issue, "Password")
}

func TestSecrets_DetectsPrivateKey(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBA..."),
	}
	check := &secretsCheck{}
	findings := check.Run(analysis)
	assert.Len(t, findings, 1)
	assert.Contains(t, findings[0].Issue, "Private key")
}

func TestSecrets_NoSecrets(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("## Rules\nAlways run tests.\nNever skip the linter."),
	}
	check := &secretsCheck{}
	assert.Empty(t, check.Run(analysis))
}

func TestSecrets_NilAnalysis(t *testing.T) {
	check := &secretsCheck{}
	assert.Nil(t, check.Run(nil))
}
