// Package static provides deterministic checks for agent specification files.
// Checks self-register via init() and produce llmutil.Finding objects identical
// to LLM output, enabling seamless merging of static and LLM results.
package static

import (
	"strings"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
)

// Check is the interface for static analysis checks on agent files.
type Check interface {
	ID() string
	Tags() []string
	Category() string
	DefaultSeverity() string
	Run(analysis *parser.AgentAnalysis) []llmutil.Finding
}

var checks []Check

// RegisterCheck adds a check to the global registry. Called from init().
func RegisterCheck(c Check) {
	checks = append(checks, c)
}

// AllChecks returns all registered checks.
func AllChecks() []Check {
	return checks
}

// ResolveChecks returns checks matching the given mode.
// Modes: "default" (default-tagged), "thorough" (all), "security" (security-tagged), "quick" (default-tagged).
func ResolveChecks(mode string) []Check {
	var resolved []Check
	for _, c := range checks {
		if matchesMode(c.Tags(), mode) {
			resolved = append(resolved, c)
		}
	}
	return resolved
}

func matchesMode(tags []string, mode string) bool {
	for _, t := range tags {
		switch mode {
		case "default", "quick":
			if t == "default" {
				return true
			}
		case "thorough":
			return true // all checks run in thorough mode
		case "security":
			if t == "security" {
				return true
			}
		}
	}
	return false
}

// RunChecks executes all checks matching the mode against the analysis.
func RunChecks(analysis *parser.AgentAnalysis, mode string) []llmutil.Finding {
	resolved := ResolveChecks(mode)
	var findings []llmutil.Finding
	for _, c := range resolved {
		findings = append(findings, c.Run(analysis)...)
	}
	return findings
}

// NormalizeContent returns content with CRLF normalized to LF.
func NormalizeContent(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

// Indicator is a continuous metric displayed to the user but not counted in structural score.
type Indicator struct {
	Name     string  // e.g., "Readability"
	Value    float64 // e.g., 22.4
	Unit     string  // e.g., "Flesch RE"
	Guidance string  // e.g., "difficult — target >30"
}

// IndicatorCheck extends Check with indicator output.
type IndicatorCheck interface {
	Check
	Indicators(analysis *parser.AgentAnalysis) []Indicator
}

// CollectIndicators runs all indicator-capable checks and returns their metrics.
func CollectIndicators(analysis *parser.AgentAnalysis, mode string) []Indicator {
	resolved := ResolveChecks(mode)
	var indicators []Indicator
	for _, c := range resolved {
		if ic, ok := c.(IndicatorCheck); ok {
			indicators = append(indicators, ic.Indicators(analysis)...)
		}
	}
	return indicators
}

// ResetChecks clears the registry. Used only in tests.
func ResetChecks() {
	checks = nil
}
