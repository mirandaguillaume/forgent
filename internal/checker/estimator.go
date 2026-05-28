package checker

import (
	"strings"

	"github.com/mirandaguillaume/forgent/internal/classifier"
)

// EstimateCompliance returns a 0.0-1.0 compliance estimate for one instruction
// on a given harness. The model is additive and transparent:
//
//   base(facet) + position_modifier + framing_modifier + verifiability_modifier
//   + harness_modifier
//
// References:
//   - Position: Liu et al. 2023 "Lost in the Middle" + Veseli et al. 2025
//   - Negation: IFEval benchmark (Zhou et al. 2023) + empirical community data
//   - Harness deltas: empirical observation, labeled as estimates
func EstimateCompliance(ir InstructionResult, harness string) float64 {
	score := baseScore(ir.Facet)
	score += positionModifier(ir.Position)
	score += framingModifier(ir.IsNegative)
	score += verifiabilityModifier(ir.IsVerifiable)
	score += harnessModifier(ir, harness)
	return clamp(score)
}

// baseScore by facet — how inherently followable each instruction type is.
func baseScore(facet classifier.Facet) float64 {
	switch facet {
	case classifier.FacetSecurity:
		return 0.82 // enforced at tool-call level, high reliability
	case classifier.FacetContext:
		return 0.78 // passive background knowledge, low violation risk
	case classifier.FacetStrategy:
		return 0.72 // behavioral, requires active application
	case classifier.FacetGuardrails:
		return 0.68 // prohibitions are harder to follow than prescriptions
	case classifier.FacetObservability:
		return 0.65 // often omitted under time pressure
	default:
		return 0.70
	}
}

// positionModifier applies the "Lost in the Middle" effect.
// Instructions in the first or last 25% of the file benefit from primacy/recency.
// Veseli et al. 2025: effect strongest when context < 50% of window.
func positionModifier(pos float64) float64 {
	if pos <= 0.25 {
		return 0.08 // primacy zone
	}
	if pos >= 0.75 {
		return 0.04 // recency zone (weaker than primacy)
	}
	return -0.06 // "lost in the middle"
}

// framingModifier penalises negative instructions.
// IFEval shows models handle "do not use X" worse than "always use Y instead".
// Empirical delta from community benchmarks: ~12-18%.
func framingModifier(isNegative bool) float64 {
	if isNegative {
		return -0.14
	}
	return 0.0
}

// verifiabilityModifier rewards instructions that can be checked statically.
// Verifiable constraints have higher compliance because they are unambiguous.
func verifiabilityModifier(isVerifiable bool) float64 {
	if isVerifiable {
		return 0.08
	}
	return 0.0
}

// harnessModifier applies per-harness adjustments.
// These are empirical estimates, clearly labeled as such.
func harnessModifier(ir InstructionResult, harness string) float64 {
	switch harness {
	case "copilot":
		delta := 0.0
		// Copilot (GPT-based by default) handles negation worse than Claude-based tools.
		if ir.IsNegative {
			delta -= 0.08
		}
		// Copilot instructions are always-on (no conditional loading like Cursor globs).
		// Long files dilute individual instruction weight.
		if ir.Position > 0.5 {
			delta -= 0.04
		}
		return delta
	case "cursor":
		// Cursor uses Claude or GPT depending on user config.
		// With Claude: similar to claude-code. We assume mixed fleet → slight discount.
		if ir.IsNegative {
			delta := -0.04
			return delta
		}
		return 0.02 // glob filtering means rules are loaded contextually → slight +
	case "claude-code":
		// Claude Code with structured CLAUDE.md: baseline, no adjustment.
		return 0.0
	default:
		return 0.0
	}
}

// isNegativeInstruction detects prohibition framing.
func isNegativeInstruction(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	return strings.HasPrefix(t, "never ") ||
		strings.HasPrefix(t, "don't ") ||
		strings.HasPrefix(t, "do not ") ||
		strings.HasPrefix(t, "avoid ") ||
		strings.HasPrefix(t, "must not") ||
		strings.Contains(t, " never ") ||
		strings.Contains(t, " do not ")
}

// isVerifiableInstruction detects statically checkable instructions.
func isVerifiableInstruction(text string, facet classifier.Facet) bool {
	t := strings.ToLower(text)
	// Security constraints are often tool-level verifiable.
	if facet == classifier.FacetSecurity {
		return true
	}
	// Syntax/format constraints are statically checkable.
	verifiableKeywords := []string{
		"tab", "space", "indent", "quote", "semicolon", "comma",
		"header", "copyright", "import", "export", "type ", "interface",
		"test file", "test suite", ".test.", "_test.", "suffix", "prefix",
	}
	for _, kw := range verifiableKeywords {
		if strings.Contains(t, kw) {
			return true
		}
	}
	return false
}

// buildSuggestions generates improvement suggestions for low-compliance instructions.
func buildSuggestions(ir InstructionResult, targets []string) []string {
	var suggestions []string

	// Suggest positive reframing for negative instructions with low scores.
	if ir.IsNegative {
		for _, h := range targets {
			if score, ok := ir.Scores[h]; ok && score < 0.60 {
				suggestions = append(suggestions,
					"Reframe as positive: \"always use X\" instead of \"never use Y\" (+~14% compliance)")
				break
			}
		}
	}

	// Suggest moving critical instructions earlier.
	if ir.Position > 0.50 && ir.Facet == classifier.FacetGuardrails {
		suggestions = append(suggestions,
			"Move to top 25% of file — guardrails in the middle lose compliance (Lost in the Middle, Liu 2023)")
	}

	return dedupe(suggestions)
}

func clamp(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

func dedupe(ss []string) []string {
	seen := make(map[string]bool)
	out := ss[:0]
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
