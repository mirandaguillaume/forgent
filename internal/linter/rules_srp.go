package linter

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/pkg/model"
)

type singleProducesOutputRule struct{}

func (r *singleProducesOutputRule) Name() string            { return "single-produces-output" }
func (r *singleProducesOutputRule) DefaultSeverity() Severity { return SeverityError }

func (r *singleProducesOutputRule) Check(skill model.SkillBehavior) *LintResult {
	count := len(skill.Context.Produces)
	if count != 1 {
		return &LintResult{
			Rule:     "single-produces-output",
			Severity: SeverityError,
			Message:  fmt.Sprintf("Skill %q must produce exactly 1 output, got %d. SRP: one skill = one deliverable.", skill.Skill, count),
			Facet:    "context",
		}
	}
	return nil
}

func init() { Register(&singleProducesOutputRule{}) }

type producesMatchesDescriptionRule struct{}

func (r *producesMatchesDescriptionRule) Name() string            { return "produces-matches-description" }
func (r *producesMatchesDescriptionRule) DefaultSeverity() Severity { return SeverityError }

func (r *producesMatchesDescriptionRule) Check(skill model.SkillBehavior) *LintResult {
	conjunctions := []string{" and ", " et ", " then ", " puis ", " & "}
	lower := strings.ToLower(skill.Strategy.Approach)
	for _, conj := range conjunctions {
		if strings.Contains(lower, conj) {
			return &LintResult{
				Rule:     "produces-matches-description",
				Severity: SeverityError,
				Message:  fmt.Sprintf("Skill %q strategy.approach contains conjunction %q suggesting multiple responsibilities. Split into separate skills.", skill.Skill, conj),
				Facet:    "strategy",
			}
		}
	}
	return nil
}

func init() { Register(&producesMatchesDescriptionRule{}) }

type skillNameMatchesOutputRule struct{}

func (r *skillNameMatchesOutputRule) Name() string            { return "skill-name-matches-output" }
func (r *skillNameMatchesOutputRule) DefaultSeverity() Severity { return SeverityError }

func (r *skillNameMatchesOutputRule) Check(skill model.SkillBehavior) *LintResult {
	patterns := []string{"-and-", "-et-", "-then-", "-puis-", "-&-"}
	lower := strings.ToLower(skill.Skill)
	for _, pat := range patterns {
		if strings.Contains(lower, pat) {
			return &LintResult{
				Rule:     "skill-name-matches-output",
				Severity: SeverityError,
				Message:  fmt.Sprintf("Skill name %q contains conjunction pattern %q. A skill name should describe a single responsibility.", skill.Skill, pat),
				Facet:    "context",
			}
		}
	}
	return nil
}

func init() { Register(&skillNameMatchesOutputRule{}) }
