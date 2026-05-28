package experiment

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	pkgsandbox "github.com/mirandaguillaume/forgent/pkg/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ─── YAML round-trip ──────────────────────────────────────────────────────────

func TestNetworkPolicy_YAMLRoundTrip(t *testing.T) {
	raw := `
id: step1
name: Step 1
prompt: do it
network_policy:
  forbidden_domains:
    - "*.internal"
    - api.stripe.com
produces:
  - key: output1
`
	var step StepDef
	require.NoError(t, yaml.Unmarshal([]byte(raw), &step))
	require.NotNil(t, step.NetworkPolicy)
	assert.Equal(t, []string{"*.internal", "api.stripe.com"}, step.NetworkPolicy.ForbiddenDomains)
	assert.Empty(t, step.NetworkPolicy.AllowedDomains)

	out, err := yaml.Marshal(step)
	require.NoError(t, err)
	var step2 StepDef
	require.NoError(t, yaml.Unmarshal(out, &step2))
	require.NotNil(t, step2.NetworkPolicy)
	assert.Equal(t, step.NetworkPolicy.ForbiddenDomains, step2.NetworkPolicy.ForbiddenDomains)
}

func TestNetworkPolicy_Absent_OmitEmpty(t *testing.T) {
	step := StepDef{ID: "x", Name: "X", Prompt: "do stuff"}
	out, err := yaml.Marshal(step)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "network_policy", "omitempty should suppress nil pointer")
}

// ─── AC6: unknown field rejection ─────────────────────────────────────────────

func TestNetworkPolicy_UnknownField_Rejected(t *testing.T) {
	raw := `
id: s1
name: S1
network_policy:
  allow_globs:
    - "*.example.com"
produces:
  - key: out
`
	var step StepDef
	err := yaml.Unmarshal([]byte(raw), &step)
	require.Error(t, err, "unknown field allow_globs should be rejected")
	assert.Contains(t, err.Error(), "allow_globs")
	assert.Contains(t, err.Error(), "allowed_domains")
}

// ─── Validate() rejection tests ───────────────────────────────────────────────

func TestValidate_RejectsBadDomainPattern_Slash(t *testing.T) {
	config := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "t1",
			Steps: []StepDef{{
				ID:   "s1",
				Name: "S1",
				NetworkPolicy: &pkgsandbox.NetworkPolicy{
					AllowedDomains: []string{"api.github.com/repos"},
				},
				Produces: []SlotSpec{{Key: "out"}},
			}},
		}},
		Provider:    &mockGuardrailProvider{responses: []string{"ok"}},
		RunsPerTask: 1,
	}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network_policy")
	assert.Contains(t, err.Error(), "no '/'")
}

func TestValidate_RejectsBadDomainPattern_QuestionMark(t *testing.T) {
	config := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "t1",
			Steps: []StepDef{{
				ID:   "s1",
				Name: "S1",
				NetworkPolicy: &pkgsandbox.NetworkPolicy{
					ForbiddenDomains: []string{"api.example.com?foo=bar"},
				},
				Produces: []SlotSpec{{Key: "out"}},
			}},
		}},
		Provider:    &mockGuardrailProvider{responses: []string{"ok"}},
		RunsPerTask: 1,
	}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network_policy")
	// '/' is NOT in this pattern — only '?' triggers the error.
	assert.Contains(t, err.Error(), "no '?'")
	assert.NotContains(t, err.Error(), "no '/'", "only '?' should be mentioned for this pattern")
}

func TestValidate_RejectsPortInPattern(t *testing.T) {
	config := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "t1",
			Steps: []StepDef{{
				ID:   "s1",
				Name: "S1",
				NetworkPolicy: &pkgsandbox.NetworkPolicy{
					AllowedDomains: []string{"api.example.com:443"},
				},
				Produces: []SlotSpec{{Key: "out"}},
			}},
		}},
		Provider:    &mockGuardrailProvider{responses: []string{"ok"}},
		RunsPerTask: 1,
	}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without port")
}

func TestValidate_RejectsBareWildcard(t *testing.T) {
	config := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "t1",
			Steps: []StepDef{{
				ID:   "s1",
				Name: "S1",
				NetworkPolicy: &pkgsandbox.NetworkPolicy{
					AllowedDomains: []string{"*"},
				},
				Produces: []SlotSpec{{Key: "out"}},
			}},
		}},
		Provider:    &mockGuardrailProvider{responses: []string{"ok"}},
		RunsPerTask: 1,
	}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "leading '*.'")
}

func TestValidate_RejectsDoubleWildcard(t *testing.T) {
	config := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "t1",
			Steps: []StepDef{{
				ID:   "s1",
				Name: "S1",
				NetworkPolicy: &pkgsandbox.NetworkPolicy{
					ForbiddenDomains: []string{"*.*.example.com"},
				},
				Produces: []SlotSpec{{Key: "out"}},
			}},
		}},
		Provider:    &mockGuardrailProvider{responses: []string{"ok"}},
		RunsPerTask: 1,
	}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "leading '*.'")
}

func TestValidate_RejectsEmptyDomainPattern(t *testing.T) {
	config := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "t1",
			Steps: []StepDef{{
				ID:   "s1",
				Name: "S1",
				NetworkPolicy: &pkgsandbox.NetworkPolicy{
					AllowedDomains: []string{"api.github.com", ""},
				},
				Produces: []SlotSpec{{Key: "out"}},
			}},
		}},
		Provider:    &mockGuardrailProvider{responses: []string{"ok"}},
		RunsPerTask: 1,
	}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty pattern")
}

func TestValidate_ValidNetworkPolicy_NoError(t *testing.T) {
	config := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "t1",
			Steps: []StepDef{{
				ID:   "s1",
				Name: "S1",
				NetworkPolicy: &pkgsandbox.NetworkPolicy{
					ForbiddenDomains: []string{"*.internal", "api.stripe.com"},
					AllowedDomains:   []string{"api.github.com"},
				},
				Produces: []SlotSpec{{Key: "out"}},
			}},
		}},
		Provider:    &mockGuardrailProvider{responses: []string{"ok"}},
		RunsPerTask: 1,
	}
	require.NoError(t, config.Validate())
}

// ─── Proxy lifecycle integration tests ───────────────────────────────────────

// mockEnvCapturingProvider records the HTTP_PROXY env var value seen during Complete().
type mockEnvCapturingProvider struct {
	capturedHTTPProxy  string
	capturedHTTPSProxy string
	capturedNoProxy    string
	response           string
}

func (m *mockEnvCapturingProvider) Complete(prompt string) (string, error) {
	m.capturedHTTPProxy = os.Getenv("HTTP_PROXY")
	m.capturedHTTPSProxy = os.Getenv("HTTPS_PROXY")
	m.capturedNoProxy = os.Getenv("NO_PROXY")
	return m.response, nil
}

func TestNetworkPolicy_PolicyPresent_ProxyEnvVarSet(t *testing.T) {
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")

	provider := &mockEnvCapturingProvider{response: "ok response"}

	task := TaskDefinition{
		ID:    "proxy-test",
		Name:  "Proxy Test",
		Input: "input data",
		Steps: []StepDef{{
			ID:     "s1",
			Name:   "Step 1",
			Prompt: "do it",
			NetworkPolicy: &pkgsandbox.NetworkPolicy{
				ForbiddenDomains: []string{"api.stripe.com"},
			},
			Produces: []SlotSpec{{Key: "output1"}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed)

	assert.True(t, strings.HasPrefix(provider.capturedHTTPProxy, "http://127.0.0.1:"),
		"HTTP_PROXY should point to local proxy during step, got: %q", provider.capturedHTTPProxy)
	// HTTPS_PROXY is intentionally NOT set: the LLM inference API uses HTTPS and
	// text+runtime conditions run in parallel goroutines — setting HTTPS_PROXY
	// process-globally would race with the text goroutine's Anthropic calls.
	assert.Empty(t, provider.capturedHTTPSProxy, "HTTPS_PROXY must not be set (parallel goroutine race risk)")
}

func TestNetworkPolicy_NoPolicy_ProxyEnvVarNotSet(t *testing.T) {
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")

	provider := &mockEnvCapturingProvider{response: "ok response"}

	task := TaskDefinition{
		ID: "no-proxy-test", Name: "No Proxy Test", Input: "input data",
		Steps: []StepDef{{
			ID: "s1", Name: "Step 1", Prompt: "do it",
			Produces: []SlotSpec{{Key: "output1"}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed)
	assert.Empty(t, provider.capturedHTTPProxy, "HTTP_PROXY must not be set when no NetworkPolicy")
	assert.Empty(t, provider.capturedHTTPSProxy)
}

func TestNetworkPolicy_ProxyEnvVarRestoredAfterStep(t *testing.T) {
	sentinel := "http://pre-existing-proxy:9999"
	t.Setenv("HTTP_PROXY", sentinel)

	provider := &mockEnvCapturingProvider{response: "ok response"}

	task := TaskDefinition{
		ID: "restore-test", Name: "Restore Test", Input: "data",
		Steps: []StepDef{{
			ID: "s1", Name: "Step 1", Prompt: "do it",
			NetworkPolicy: &pkgsandbox.NetworkPolicy{ForbiddenDomains: []string{"evil.example.com"}},
			Produces:      []SlotSpec{{Key: "output1"}},
		}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks: []TaskDefinition{task}, Provider: provider, RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.NotEqual(t, sentinel, provider.capturedHTTPProxy, "proxy URL should override sentinel during step")
	assert.True(t, strings.HasPrefix(provider.capturedHTTPProxy, "http://127.0.0.1:"))
	// t.Setenv restores the value after the test, but verify it was restored after the step.
	assert.Equal(t, sentinel, os.Getenv("HTTP_PROXY"), "HTTP_PROXY must be restored to sentinel after step")
}

// ─── T4.3: audit wiring — Action="network" in AccessEntry ─────────────────────

func TestNetworkPolicy_AuditWiring_NetworkActionRecorded(t *testing.T) {
	// This test verifies that the AuditFunc passed to NewProxyServer correctly
	// writes an AccessEntry{Action:"network"} to the sandbox log when a domain
	// is denied. We drive the proxy directly (not via RunRuntimeExperiment) so
	// we can make real HTTP requests through it.
	var log []AccessEntry
	auditFn := pkgsandbox.AuditFunc(func(host string, allowed bool, reason string) {
		log = append(log, AccessEntry{
			StepID:  "test-step",
			Action:  "network",
			Path:    host,
			Allowed: allowed,
			Reason:  reason,
		})
	})

	policy := &pkgsandbox.NetworkPolicy{ForbiddenDomains: []string{"blocked.example.com"}}
	proxy, err := pkgsandbox.NewProxyServer(policy, auditFn)
	require.NoError(t, err)
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL())
	require.NoError(t, err)

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}
	resp, err := client.Get("http://blocked.example.com/")
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	require.Len(t, log, 1)
	assert.Equal(t, "network", log[0].Action)
	assert.Equal(t, "blocked.example.com", log[0].Path)
	assert.False(t, log[0].Allowed)
	assert.Contains(t, log[0].Reason, "blocked.example.com")
}
