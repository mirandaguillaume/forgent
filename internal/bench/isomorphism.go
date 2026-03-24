package bench

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SkillSignature captures a skill's name and I/O contract.
type SkillSignature struct {
	Name     string
	Consumes []string
	Produces []string
}

// IsomorphismResult holds the outcome of the cross-target isomorphism benchmark.
type IsomorphismResult struct {
	SkillNamesMatch  bool
	IOContractsMatch bool
	StructureScore   float64 // 0.0 - 1.0
	ClaudeSkills     []SkillSignature
	CopilotSkills    []SkillSignature
}

var (
	// Parse I/O from frontmatter description line produced by BuildSkillDescription:
	// "description: analytical-based skill consuming source-code, diff to produce review-report"
	descConsumesRe = regexp.MustCompile(`(?m)^description:.*consuming\s+(.+?)(?:\s+to produce|$)`)
	descProducesRe = regexp.MustCompile(`(?m)^description:.*to produce\s+(.+)$`)
)

// RunIsomorphism builds for both claude and copilot targets, then compares
// the extracted skill signatures (φ function from P9).
func RunIsomorphism(skillsDir, agentsDir string) (*IsomorphismResult, error) {
	claudeDir, err := os.MkdirTemp("", "bench-iso-claude-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(claudeDir)

	copilotDir, err := os.MkdirTemp("", "bench-iso-copilot-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(copilotDir)

	if err := buildToDir(skillsDir, agentsDir, claudeDir, "claude"); err != nil {
		return nil, err
	}
	if err := buildToDir(skillsDir, agentsDir, copilotDir, "copilot"); err != nil {
		return nil, err
	}

	claudeSkills, err := extractSignatures(claudeDir)
	if err != nil {
		return nil, err
	}
	copilotSkills, err := extractSignatures(copilotDir)
	if err != nil {
		return nil, err
	}

	// Sort for deterministic comparison
	sortSigs(claudeSkills)
	sortSigs(copilotSkills)

	namesMatch := sigNamesEqual(claudeSkills, copilotSkills)
	ioMatch := sigIOEqual(claudeSkills, copilotSkills)

	// Structure score: compare 3 dimensions (names, consumes, produces)
	matched := 0
	total := 3
	if namesMatch {
		matched++
	}
	if ioMatch {
		matched++
	}
	// Check execution order from agent files
	claudeOrder := extractSkillOrder(claudeDir)
	copilotOrder := extractSkillOrder(copilotDir)
	if sliceEqual(claudeOrder, copilotOrder) {
		matched++
	}

	return &IsomorphismResult{
		SkillNamesMatch:  namesMatch,
		IOContractsMatch: ioMatch,
		StructureScore:   float64(matched) / float64(total),
		ClaudeSkills:     claudeSkills,
		CopilotSkills:    copilotSkills,
	}, nil
}

// extractSignatures walks the generated skills directory and parses
// the Consumes/Produces lines from each skill markdown file.
func extractSignatures(outputDir string) ([]SkillSignature, error) {
	var sigs []SkillSignature

	// Find skill files — look for SKILL.md files in subdirectories
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if filepath.Base(path) != "SKILL.md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)

		// Extract skill name from parent directory
		name := filepath.Base(filepath.Dir(path))

		sig := SkillSignature{Name: name}

		if m := descConsumesRe.FindStringSubmatch(content); len(m) > 1 {
			sig.Consumes = parseCSV(m[1])
		}
		if m := descProducesRe.FindStringSubmatch(content); len(m) > 1 {
			sig.Produces = parseCSV(m[1])
		}

		sigs = append(sigs, sig)
		return nil
	})
	return sigs, err
}

// extractSkillOrder reads the agent file and extracts the skill execution order.
func extractSkillOrder(outputDir string) []string {
	var order []string
	stepRe := regexp.MustCompile(`(?m)^### Step \d+: (.+)$`)

	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		// Agent files are .md files in the agents/ directory
		rel, _ := filepath.Rel(outputDir, path)
		if !strings.Contains(rel, "agent") || filepath.Base(path) == "SKILL.md" {
			return nil
		}
		data, _ := os.ReadFile(path)
		matches := stepRe.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			order = append(order, strings.TrimSpace(m[1]))
		}
		return nil
	})
	return order
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	sort.Strings(result)
	return result
}

func sortSigs(sigs []SkillSignature) {
	sort.Slice(sigs, func(i, j int) bool {
		return sigs[i].Name < sigs[j].Name
	})
}

func sigNamesEqual(a, b []SkillSignature) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name {
			return false
		}
	}
	return true
}

func sigIOEqual(a, b []SkillSignature) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sliceEqual(a[i].Consumes, b[i].Consumes) || !sliceEqual(a[i].Produces, b[i].Produces) {
			return false
		}
	}
	return true
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
