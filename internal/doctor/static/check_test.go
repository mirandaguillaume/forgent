package static

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/stretchr/testify/assert"
)

type mockCheck struct {
	id       string
	tags     []string
	category string
	severity string
	findings []llmutil.Finding
}

func (m *mockCheck) ID() string                                    { return m.id }
func (m *mockCheck) Tags() []string                                { return m.tags }
func (m *mockCheck) Category() string                              { return m.category }
func (m *mockCheck) DefaultSeverity() string                       { return m.severity }
func (m *mockCheck) Run(_ *parser.AgentAnalysis) []llmutil.Finding { return m.findings }

func TestRegisterAndResolve(t *testing.T) {
	ResetChecks()
	defer ResetChecks()

	RegisterCheck(&mockCheck{id: "default-check", tags: []string{"default"}, findings: []llmutil.Finding{{Issue: "a"}}})
	RegisterCheck(&mockCheck{id: "thorough-check", tags: []string{"thorough"}, findings: []llmutil.Finding{{Issue: "b"}}})
	RegisterCheck(&mockCheck{id: "security-check", tags: []string{"security"}, findings: []llmutil.Finding{{Issue: "c"}}})
	RegisterCheck(&mockCheck{id: "default-security", tags: []string{"default", "security"}, findings: []llmutil.Finding{{Issue: "d"}}})

	assert.Len(t, AllChecks(), 4)

	// Default mode: only default-tagged
	defaultChecks := ResolveChecks("default")
	assert.Len(t, defaultChecks, 2) // default-check + default-security
	assert.Equal(t, "default-check", defaultChecks[0].ID())
	assert.Equal(t, "default-security", defaultChecks[1].ID())

	// Quick = same as default
	quickChecks := ResolveChecks("quick")
	assert.Len(t, quickChecks, 2)

	// Thorough: all checks
	thoroughChecks := ResolveChecks("thorough")
	assert.Len(t, thoroughChecks, 4)

	// Security: only security-tagged
	securityChecks := ResolveChecks("security")
	assert.Len(t, securityChecks, 2) // security-check + default-security
}

func TestRunChecks(t *testing.T) {
	ResetChecks()
	defer ResetChecks()

	RegisterCheck(&mockCheck{id: "a", tags: []string{"default"}, findings: []llmutil.Finding{{Category: "test", Issue: "finding-a"}}})
	RegisterCheck(&mockCheck{id: "b", tags: []string{"thorough"}, findings: []llmutil.Finding{{Category: "test", Issue: "finding-b"}}})

	// Default mode
	findings := RunChecks(&parser.AgentAnalysis{}, "default")
	assert.Len(t, findings, 1)
	assert.Equal(t, "finding-a", findings[0].Issue)

	// Thorough mode
	findings = RunChecks(&parser.AgentAnalysis{}, "thorough")
	assert.Len(t, findings, 2)
}

func TestEmptyRegistry(t *testing.T) {
	ResetChecks()
	defer ResetChecks()

	assert.Empty(t, AllChecks())
	assert.Empty(t, ResolveChecks("default"))
	assert.Empty(t, RunChecks(&parser.AgentAnalysis{}, "default"))
}
