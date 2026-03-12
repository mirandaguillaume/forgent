package linter

import (
	"fmt"

	"github.com/mirandaguillaume/forgent/pkg/model"
)

// Severity represents the severity level of a lint result.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// LintResult represents a single lint finding.
type LintResult struct {
	Rule     string
	Severity Severity
	Message  string
	Facet    string
}

type lintRule func(skill model.SkillBehavior) *LintResult

func noEmptyTools(skill model.SkillBehavior) *LintResult {
	if len(skill.Strategy.Tools) == 0 {
		return &LintResult{
			Rule:     "no-empty-tools",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("Skill %q has no tools defined. An agent without tools has limited capability.", skill.Skill),
			Facet:    "strategy",
		}
	}
	return nil
}

func hasGuardrails(skill model.SkillBehavior) *LintResult {
	if len(skill.Guardrails) == 0 {
		return &LintResult{
			Rule:     "has-guardrails",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("Skill %q has no guardrails. Consider adding limits (timeout, max_tokens, etc.).", skill.Skill),
			Facet:    "guardrails",
		}
	}
	return nil
}

func observableOutputs(skill model.SkillBehavior) *LintResult {
	if len(skill.Context.Produces) > 0 && len(skill.Observability.Metrics) == 0 {
		return &LintResult{
			Rule:     "observable-outputs",
			Severity: SeverityWarning,
			Message:  fmt.Sprintf("Skill %q produces data but has no observability metrics. Add metrics to track output quality.", skill.Skill),
			Facet:    "observability",
		}
	}
	return nil
}

func securityNeedsGuardrails(skill model.SkillBehavior) *LintResult {
	hasHighAccess := skill.Security.Filesystem == model.AccessFull || skill.Security.Filesystem == model.AccessReadWrite
	if hasHighAccess && len(skill.Guardrails) == 0 {
		return &LintResult{
			Rule:     "security-needs-guardrails",
			Severity: SeverityError,
			Message:  fmt.Sprintf("Skill %q has %s filesystem access but no guardrails. This is dangerous.", skill.Skill, skill.Security.Filesystem),
			Facet:    "security",
		}
	}
	return nil
}

func hasWhenToUse(skill model.SkillBehavior) *LintResult {
	if skill.WhenToUse.IsEmpty() {
		return &LintResult{
			Rule:     "has-when-to-use",
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("Skill %q has no when_to_use guidance. Consider adding triggers and boundaries.", skill.Skill),
			Facet:    "when_to_use",
		}
	}
	return nil
}

var allRules = []lintRule{
	noEmptyTools,
	hasGuardrails,
	observableOutputs,
	securityNeedsGuardrails,
	hasWhenToUse,
}

// LintSkill runs all lint rules against a skill and returns findings.
func LintSkill(skill model.SkillBehavior) []LintResult {
	var results []LintResult
	for _, rule := range allRules {
		if r := rule(skill); r != nil {
			results = append(results, *r)
		}
	}
	return results
}
