package forgent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"
)

func init() {
	spec.Register("forgent", func() spec.Generator {
		return &forgentGenerator{}
	})
}

type forgentGenerator struct{}

func (g *forgentGenerator) Target() string          { return "forgent" }
func (g *forgentGenerator) DefaultOutputDir() string { return ".forgent" }
func (g *forgentGenerator) ContextDir() string       { return "" }

// AgentPath returns the relative path for the generated main.go within the output dir.
func (g *forgentGenerator) AgentPath(name string) string {
	safe := safeAgentName(name)
	return filepath.Join(safe, "main.go")
}

// GenerateAgent returns the Go main.go source code for the agent's DAG runtime.
// It also writes the go.mod file as a side effect (the builder only writes AgentPath).
func (g *forgentGenerator) GenerateAgent(agent model.AgentComposition, skills []model.SkillBehavior, outputDir string) string {
	safe := safeAgentName(agent.Agent)
	modDir := filepath.Join(outputDir, safe)
	if err := os.MkdirAll(modDir, 0755); err == nil {
		modContent := GenerateGoMod(safe)
		_ = os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(modContent), 0644)
		_ = os.WriteFile(filepath.Join(modDir, "go.sum"), []byte(""), 0644)
	}
	return GenerateAgentGo(agent, skills)
}

// safeAgentName converts an agent name to a filesystem/module-safe identifier.
func safeAgentName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}
