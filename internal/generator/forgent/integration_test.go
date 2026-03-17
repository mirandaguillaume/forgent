package forgent_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mirandaguillaume/forgent/internal/generator/forgent"
)

func TestIntegration_GeneratedCodeCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	gen, err := spec.Get("forgent")
	require.NoError(t, err)
	ag := gen.(spec.AgentGenerator)

	agent := model.AgentComposition{
		Agent:    "test-agent",
		Skills:   []string{"analyzer"},
		Consumes: []string{"input_data"},
		Produces: []string{"output_data"},
	}
	skills := []model.SkillBehavior{
		{
			Skill:   "analyzer",
			Version: "1.0.0",
			Context: model.ContextFacet{
				Consumes: []string{"input_data"},
				Produces: []string{"output_data"},
			},
			Strategy: model.StrategyFacet{
				Approach: "analysis",
				Steps:    []string{"analyze the input", "produce output"},
			},
			Security: model.SecurityFacet{
				Filesystem: model.AccessReadOnly,
				Network:    model.NetworkNone,
			},
		},
	}

	tmpDir := t.TempDir()
	code := ag.GenerateAgent(agent, skills, tmpDir)

	// Write main.go
	agentDir := filepath.Join(tmpDir, "test_agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "main.go"), []byte(code), 0644))

	// Write go.mod pointing to actual repo root
	repoRoot, err := filepath.Abs("../../..")
	require.NoError(t, err)
	goMod := "module test_agent\n\ngo 1.22\n\nrequire github.com/mirandaguillaume/forgent v0.0.0\n\nreplace github.com/mirandaguillaume/forgent => " + repoRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "go.mod"), []byte(goMod), 0644))

	// go mod tidy
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = agentDir
	tidyOut, _ := tidy.CombinedOutput()

	// go build
	build := exec.Command("go", "build", ".")
	build.Dir = agentDir
	buildOut, buildErr := build.CombinedOutput()

	assert.NoError(t, buildErr, "generated code must compile:\ngo mod tidy: %s\ngo build: %s", string(tidyOut), string(buildOut))
}
