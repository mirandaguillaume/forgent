package claude

import (
	"fmt"
	"github.com/mirandaguillaume/forgent/pkg/model"
)

func PrintStagedOutput() {
	agent := model.AgentComposition{
		Agent:       "code-reviewer",
		Description: "Multi-stage code review pipeline",
		Stages: []model.Stage{
			{Name: "preflight", Strategy: model.OrchestrationSequential, Skills: []string{"eligibility-checker", "summarizer"}},
			{Name: "analysis", Strategy: model.OrchestrationParallel, Skills: []string{"bug-scanner", "history-reviewer"}},
			{Name: "publish", Strategy: model.OrchestrationSequential, Skills: []string{"commenter"}},
		},
	}

	skills := []model.SkillBehavior{
		{
			Skill:    "eligibility-checker",
			Context:  model.ContextFacet{Consumes: []string{"pr_url"}, Produces: []string{"eligibility_status"}, Memory: model.MemoryShortTerm},
			Strategy: model.StrategyFacet{Approach: "gate-check", Tools: []string{"bash"}, Effort: model.EffortLight},
			Security: model.SecurityFacet{Filesystem: model.AccessNone, Network: model.NetworkAllowlist},
		},
		{
			Skill:    "summarizer",
			Context:  model.ContextFacet{Consumes: []string{"pr_url"}, Produces: []string{"pr_summary"}, Memory: model.MemoryShortTerm},
			Strategy: model.StrategyFacet{Approach: "diff-first", Tools: []string{"bash"}, Effort: model.EffortLight},
			Security: model.SecurityFacet{Filesystem: model.AccessNone, Network: model.NetworkAllowlist},
		},
		{
			Skill:    "bug-scanner",
			Context:  model.ContextFacet{Consumes: []string{"pr_diff"}, Produces: []string{"review_issues"}, Memory: model.MemoryShortTerm},
			Strategy: model.StrategyFacet{Approach: "diff-first", Tools: []string{"read_file"}, Effort: model.EffortMedium},
			Security: model.SecurityFacet{Filesystem: model.AccessReadOnly, Network: model.NetworkNone},
		},
		{
			Skill:    "history-reviewer",
			Context:  model.ContextFacet{Consumes: []string{"pr_diff", "git_blame"}, Produces: []string{"review_issues"}, Memory: model.MemoryShortTerm},
			Strategy: model.StrategyFacet{Approach: "history-first", Tools: []string{"bash", "read_file"}, Effort: model.EffortMedium},
			Security: model.SecurityFacet{Filesystem: model.AccessReadOnly, Network: model.NetworkNone},
		},
		{
			Skill:    "commenter",
			Context:  model.ContextFacet{Consumes: []string{"scored_issues", "pr_url"}, Produces: []string{"review_comment"}, Memory: model.MemoryShortTerm},
			Strategy: model.StrategyFacet{Approach: "output-format", Tools: []string{"bash"}, Effort: model.EffortLight},
			Security: model.SecurityFacet{Filesystem: model.AccessNone, Network: model.NetworkAllowlist},
		},
	}

	output := GenerateAgentMd(agent, skills, ".claude")
	fmt.Println(output)
}
