package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/generator"
	yamlloader "github.com/mirandaguillaume/forgent/internal/yaml"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"

	// Register generators
	_ "github.com/mirandaguillaume/forgent/internal/generator/claude"
	_ "github.com/mirandaguillaume/forgent/internal/generator/copilot"
)

// TokenResult holds the outcome of the token overhead benchmark.
type TokenResult struct {
	ComposedWords   int
	MonolithicWords int
	OverheadPct     float64
	ComposedFiles   int
}

// RunTokenOverhead measures the word count overhead of composed vs monolithic artifacts.
func RunTokenOverhead(skillsDir, agentsDir, target string) (*TokenResult, error) {
	tmpDir, err := os.MkdirTemp("", "bench-token-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// Build composed artifacts using the generator directly (avoids cmd import cycle)
	if err := buildToDir(skillsDir, agentsDir, tmpDir, target); err != nil {
		return nil, err
	}

	// Count words in all generated files
	composedWords := 0
	composedFiles := 0
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		composedWords += generator.CountWords(string(data))
		composedFiles++
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build monolithic equivalent: concatenate raw skill content without structure
	monolithicWords, err := countMonolithic(skillsDir, agentsDir)
	if err != nil {
		return nil, err
	}

	overheadPct := 0.0
	if monolithicWords > 0 {
		overheadPct = float64(composedWords-monolithicWords) / float64(monolithicWords) * 100
	}

	return &TokenResult{
		ComposedWords:   composedWords,
		MonolithicWords: monolithicWords,
		OverheadPct:     overheadPct,
		ComposedFiles:   composedFiles,
	}, nil
}

// buildToDir is a minimal build pipeline for benchmarks.
// It avoids importing internal/cmd (which would create a cycle) by calling
// the spec/generator packages directly.
func buildToDir(skillsDir, agentsDir, outputDir, target string) error {
	return buildToDirWithOptions(skillsDir, agentsDir, outputDir, target, false)
}

// buildToDirCompact builds with compact mode enabled (skills inlined in agent).
func buildToDirCompact(skillsDir, agentsDir, outputDir, target string) error {
	return buildToDirWithOptions(skillsDir, agentsDir, outputDir, target, true)
}

func buildToDirWithOptions(skillsDir, agentsDir, outputDir, target string, compact bool) error {
	gen, err := spec.Get(target)
	if err != nil {
		return err
	}

	// Apply compact option if supported
	if c, ok := gen.(spec.Configurable); ok {
		c.SetOptions(spec.GeneratorOptions{Compact: compact})
	}

	// Parse skills
	skills, skillMap, err := parseSkills(skillsDir)
	if err != nil {
		return err
	}

	// Generate skill files (skip when compact — they're inlined in the agent)
	sg, ok := gen.(spec.SkillGenerator)
	if ok && !compact {
		for _, skill := range skills {
			md := sg.GenerateSkill(skill)
			relPath := sg.SkillPath(skill.Skill)
			fullPath := filepath.Join(outputDir, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fullPath, []byte(md), 0644); err != nil {
				return err
			}
		}
	}

	// Parse and generate agents
	ag, hasAG := gen.(spec.AgentGenerator)
	if hasAG {
		agents, err := parseAgents(agentsDir)
		if err != nil {
			return err
		}
		for _, agent := range agents {
			var agentSkills []model.SkillBehavior
			for _, sName := range agent.Skills {
				if s, ok := skillMap[sName]; ok {
					agentSkills = append(agentSkills, s)
				}
			}
			md := ag.GenerateAgent(agent, agentSkills, outputDir)
			relPath := ag.AgentPath(agent.Agent)
			fullPath := filepath.Join(outputDir, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fullPath, []byte(md), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseSkills reads all .skill.yaml files from the directory.
func parseSkills(dir string) ([]model.SkillBehavior, map[string]model.SkillBehavior, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var skills []model.SkillBehavior
	skillMap := make(map[string]model.SkillBehavior)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".skill.yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		skill, err := yamlloader.ParseSkillYAML(string(data))
		if err != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		skills = append(skills, skill)
		skillMap[skill.Skill] = skill
	}
	return skills, skillMap, nil
}

// parseAgents reads all .agent.yaml files from the directory.
func parseAgents(dir string) ([]model.AgentComposition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var agents []model.AgentComposition
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".agent.yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		agent, err := yamlloader.ParseAgentYAML(string(data))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		agents = append(agents, agent)
	}
	return agents, nil
}

// countMonolithic builds a flat text equivalent of a monolithic prompt.
// It includes all semantic content from skills and agents but without
// markdown structural overhead (no ## headers, ---, frontmatter, bold labels).
func countMonolithic(skillsDir, agentsDir string) (int, error) {
	var sb strings.Builder

	skills, _, err := parseSkills(skillsDir)
	if err != nil {
		return 0, err
	}
	for _, skill := range skills {
		// Skill identity and description
		sb.WriteString(skill.Skill + " ")
		sb.WriteString(generator.BuildSkillDescription(skill) + " ")

		// Strategy content
		sb.WriteString(skill.Strategy.Approach + " ")
		sb.WriteString(strings.Join(skill.Strategy.Tools, " ") + " ")
		for _, step := range skill.Strategy.Steps {
			sb.WriteString(step + " ")
		}

		// Context
		sb.WriteString(strings.Join(skill.Context.Consumes, " ") + " ")
		sb.WriteString(strings.Join(skill.Context.Produces, " ") + " ")
		sb.WriteString(string(skill.Context.Memory) + " ")

		// Guardrails
		for _, g := range skill.Guardrails {
			if sv, ok := g.StringValue(); ok {
				sb.WriteString(sv + " ")
			} else if mv, ok := g.MapValue(); ok {
				for k, v := range mv {
					sb.WriteString(fmt.Sprintf("%s %v ", k, v))
				}
			}
		}

		// Security
		sb.WriteString(fmt.Sprintf("%s %s ", skill.Security.Filesystem, skill.Security.Network))

		// Observability
		sb.WriteString(string(skill.Observability.TraceLevel) + " ")
		sb.WriteString(strings.Join(skill.Observability.Metrics, " ") + " ")
	}

	agents, err := parseAgents(agentsDir)
	if err != nil {
		return 0, err
	}
	for _, agent := range agents {
		sb.WriteString(agent.Agent + " ")
		sb.WriteString(agent.Description + " ")
		sb.WriteString(strings.Join(agent.Skills, " ") + " ")
		sb.WriteString(string(agent.Orchestration) + " ")
		sb.WriteString(strings.Join(agent.Consumes, " ") + " ")
		sb.WriteString(strings.Join(agent.Produces, " ") + " ")
	}

	return generator.CountWords(sb.String()), nil
}

// RunTokenOverheadCompact measures token overhead with compact mode enabled.
func RunTokenOverheadCompact(skillsDir, agentsDir, target string) (*TokenResult, error) {
	tmpDir, err := os.MkdirTemp("", "bench-token-compact-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := buildToDirCompact(skillsDir, agentsDir, tmpDir, target); err != nil {
		return nil, err
	}

	composedWords := 0
	composedFiles := 0
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		composedWords += generator.CountWords(string(data))
		composedFiles++
		return nil
	})
	if err != nil {
		return nil, err
	}

	monolithicWords, err := countMonolithic(skillsDir, agentsDir)
	if err != nil {
		return nil, err
	}

	overheadPct := 0.0
	if monolithicWords > 0 {
		overheadPct = float64(composedWords-monolithicWords) / float64(monolithicWords) * 100
	}

	return &TokenResult{
		ComposedWords:   composedWords,
		MonolithicWords: monolithicWords,
		OverheadPct:     overheadPct,
		ComposedFiles:   composedFiles,
	}, nil
}

type benchError struct {
	msg string
}

func (e *benchError) Error() string { return e.msg }
