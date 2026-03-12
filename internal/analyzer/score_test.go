package analyzer

import (
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func makeFullSkill(overrides ...func(*model.SkillBehavior)) model.SkillBehavior {
	timeoutGR := model.GuardrailRule{}
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "timeout", Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: "5min", Tag: "!!str"},
		},
	}
	_ = timeoutGR.UnmarshalYAML(node)

	maxRetriesGR := model.GuardrailRule{}
	strNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "max_retries: 3", Tag: "!!str"}
	_ = maxRetriesGR.UnmarshalYAML(strNode)

	skill := model.SkillBehavior{
		Skill:   "test-skill",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"input"},
			Produces: []string{"output"},
			Memory:   model.MemoryConversation,
		},
		Strategy: model.StrategyFacet{
			Tools:    []string{"read_file", "grep"},
			Approach: "diff-first",
			Steps:    []string{"step1", "step2"},
		},
		Guardrails: []model.GuardrailRule{timeoutGR, maxRetriesGR},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelDetailed,
			Metrics:    []string{"tokens", "latency"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadOnly,
			Network:    model.NetworkNone,
			Secrets:    []string{},
		},
	}

	for _, fn := range overrides {
		fn(&skill)
	}
	return skill
}

func makeFullAgent(overrides ...func(*model.AgentComposition)) model.AgentComposition {
	agent := model.AgentComposition{
		Agent:         "test-agent",
		Skills:        []string{"skill-a", "skill-b"},
		Orchestration: model.OrchestrationSequential,
		Description:   "A well-described agent",
	}
	for _, fn := range overrides {
		fn(&agent)
	}
	return agent
}

func TestScoreSkill_HighScore(t *testing.T) {
	result := ScoreSkill(makeFullSkill())
	assert.GreaterOrEqual(t, result.Total, 80)
}

func TestScoreSkill_PenalizeMissingTools(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	noTools := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Strategy = model.StrategyFacet{Tools: []string{}, Approach: "seq", Steps: []string{"s1"}}
	}))
	assert.Less(t, noTools.Total, full.Total)
	assert.Less(t, noTools.Breakdown.Strategy, full.Breakdown.Strategy)
}

func TestScoreSkill_PenalizeMissingGuardrails(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	noGR := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Guardrails = nil
	}))
	assert.Less(t, noGR.Total, full.Total)
	assert.Equal(t, 0, noGR.Breakdown.Guardrails)
}

func TestScoreSkill_PenalizeMissingSteps(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	noSteps := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Strategy = model.StrategyFacet{Tools: []string{"read_file"}, Approach: "seq", Steps: []string{}}
	}))
	assert.Less(t, noSteps.Breakdown.Strategy, full.Breakdown.Strategy)
}

func TestScoreSkill_PenalizeMissingObservability(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	noObs := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Observability = model.ObservabilityFacet{TraceLevel: model.TraceLevelMinimal, Metrics: []string{}}
	}))
	assert.Less(t, noObs.Breakdown.Observability, full.Breakdown.Observability)
}

func TestScoreSkill_RewardRestrictiveSecurity(t *testing.T) {
	readOnly := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Security = model.SecurityFacet{Filesystem: model.AccessReadOnly, Network: model.NetworkNone, Secrets: []string{}}
	}))
	fullAccess := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Security = model.SecurityFacet{Filesystem: model.AccessFull, Network: model.NetworkFull, Secrets: []string{"API_KEY"}}
	}))
	assert.Greater(t, readOnly.Breakdown.Security, fullAccess.Breakdown.Security)
}

func TestScoreSkill_PenalizeEmptyContext(t *testing.T) {
	full := ScoreSkill(makeFullSkill())
	emptyCtx := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{}, Memory: model.MemoryShortTerm}
	}))
	assert.Less(t, emptyCtx.Breakdown.Context, full.Breakdown.Context)
}

func TestScoreSkill_BreakdownFields(t *testing.T) {
	result := ScoreSkill(makeFullSkill())
	assert.Greater(t, result.Breakdown.Context, 0)
	assert.Greater(t, result.Breakdown.Strategy, 0)
	assert.Greater(t, result.Breakdown.Guardrails, 0)
	assert.Greater(t, result.Breakdown.Observability, 0)
	assert.Greater(t, result.Breakdown.Security, 0)
}

func TestScoreSkill_TotalBetween0And100(t *testing.T) {
	result := ScoreSkill(makeFullSkill())
	assert.GreaterOrEqual(t, result.Total, 0)
	assert.LessOrEqual(t, result.Total, 100)
}

func TestScoreSkill_WorstCase(t *testing.T) {
	worst := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{}, Memory: model.MemoryShortTerm}
		s.Strategy = model.StrategyFacet{Tools: []string{}, Approach: "", Steps: []string{}}
		s.Guardrails = nil
		s.Observability = model.ObservabilityFacet{TraceLevel: model.TraceLevelMinimal, Metrics: []string{}}
		s.Security = model.SecurityFacet{Filesystem: model.AccessFull, Network: model.NetworkFull, Secrets: []string{"SECRET"}}
	}))
	assert.Less(t, worst.Total, 30)
}

func TestScoreAgent_HighScore(t *testing.T) {
	skills := []model.SkillBehavior{
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-a"
			s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{"data"}, Memory: model.MemoryShortTerm}
		}),
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-b"
			s.Context = model.ContextFacet{Consumes: []string{"data"}, Produces: []string{"result"}, Memory: model.MemoryShortTerm}
		}),
	}
	result := ScoreAgent(makeFullAgent(), skills)
	assert.GreaterOrEqual(t, result.Total, 80)
}

func TestScoreAgent_PenalizeMissingDescription(t *testing.T) {
	withDesc := ScoreAgent(makeFullAgent(), nil)
	noDesc := ScoreAgent(makeFullAgent(func(a *model.AgentComposition) {
		a.Description = ""
	}), nil)
	assert.Less(t, noDesc.Total, withDesc.Total)
}

func TestScoreAgent_PenalizeBrokenDataFlow(t *testing.T) {
	goodFlow := []model.SkillBehavior{
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-a"
			s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{"data"}, Memory: model.MemoryShortTerm}
		}),
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-b"
			s.Context = model.ContextFacet{Consumes: []string{"data"}, Produces: []string{"result"}, Memory: model.MemoryShortTerm}
		}),
	}
	brokenFlow := []model.SkillBehavior{
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-a"
			s.Context = model.ContextFacet{Consumes: []string{"data"}, Produces: []string{"result"}, Memory: model.MemoryShortTerm}
		}),
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-b"
			s.Context = model.ContextFacet{Consumes: []string{}, Produces: []string{"data"}, Memory: model.MemoryShortTerm}
		}),
	}
	good := ScoreAgent(makeFullAgent(), goodFlow)
	broken := ScoreAgent(makeFullAgent(), brokenFlow)
	assert.Less(t, broken.Breakdown.DataFlow, good.Breakdown.DataFlow)
}

func TestScoreAgent_EnvironmentInputsNotPenalized(t *testing.T) {
	skills := []model.SkillBehavior{
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-a"
			s.Context = model.ContextFacet{Consumes: []string{"file_tree", "source_code"}, Produces: []string{"lint_results"}, Memory: model.MemoryShortTerm}
		}),
		makeFullSkill(func(s *model.SkillBehavior) {
			s.Skill = "skill-b"
			s.Context = model.ContextFacet{Consumes: []string{"file_tree", "lint_results"}, Produces: []string{"report"}, Memory: model.MemoryShortTerm}
		}),
	}
	result := ScoreAgent(makeFullAgent(), skills)
	// lint_results is the only inter-skill dependency and it's correctly ordered
	assert.Equal(t, 35, result.Breakdown.DataFlow)
}

func TestScoreAgent_PenalizeSingleSkill(t *testing.T) {
	multi := ScoreAgent(makeFullAgent(func(a *model.AgentComposition) {
		a.Skills = []string{"a", "b"}
	}), nil)
	single := ScoreAgent(makeFullAgent(func(a *model.AgentComposition) {
		a.Skills = []string{"a"}
	}), nil)
	assert.Less(t, single.Breakdown.Composition, multi.Breakdown.Composition)
}

func TestScoreAgent_TotalBetween0And100(t *testing.T) {
	result := ScoreAgent(makeFullAgent(), nil)
	assert.GreaterOrEqual(t, result.Total, 0)
	assert.LessOrEqual(t, result.Total, 100)
}

func TestScoreSkill_WhenToUseBonus(t *testing.T) {
	without := ScoreSkill(makeFullSkill())
	with := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.WhenToUse = model.WhenToUseFacet{
			Triggers: []string{"bugs"},
			DontUse:  []string{"typos"},
			Especially: []string{"pressure"},
		}
	}))
	assert.Greater(t, with.Total, without.Total)
	assert.Equal(t, weightWhenToUse, with.Breakdown.WhenToUse)
	assert.Equal(t, 0, without.Breakdown.WhenToUse)
}

func TestScoreSkill_AntiPatternsBonus(t *testing.T) {
	without := ScoreSkill(makeFullSkill())
	with := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.AntiPatterns = []model.AntiPattern{
			{Excuse: "a", Reality: "b"},
			{Excuse: "c", Reality: "d"},
		}
	}))
	assert.Greater(t, with.Total, without.Total)
	assert.Equal(t, weightAntiPatterns, with.Breakdown.AntiPatterns)
	assert.Equal(t, 0, without.Breakdown.AntiPatterns)
}

func TestScoreSkill_ExamplesBonus(t *testing.T) {
	without := ScoreSkill(makeFullSkill())
	with := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Examples = []model.CodeExample{
			{Label: "good", Code: "test", Lang: "bash"},
			{Label: "bad", Code: "skip"},
		}
	}))
	assert.Greater(t, with.Total, without.Total)
	assert.Equal(t, weightExamples, with.Breakdown.Examples)
	assert.Equal(t, 0, without.Breakdown.Examples)
}

func TestScoreSkill_MaxWithAllFacets(t *testing.T) {
	result := ScoreSkill(makeFullSkill(func(s *model.SkillBehavior) {
		s.Strategy.Steps = []string{"s1", "s2", "s3"}
		s.WhenToUse = model.WhenToUseFacet{
			Triggers: []string{"bugs"}, DontUse: []string{"typos"}, Especially: []string{"pressure"},
		}
		s.AntiPatterns = []model.AntiPattern{{Excuse: "a", Reality: "b"}, {Excuse: "c", Reality: "d"}}
		s.Examples = []model.CodeExample{{Label: "g", Code: "c"}, {Label: "b", Code: "d"}}
	}))
	assert.LessOrEqual(t, result.Total, 100)
}

// --- Calibration: superpowers skills mapped to SkillBehavior ---

func makeGuardrail(s string) model.GuardrailRule {
	var g model.GuardrailRule
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: s}
	_ = g.UnmarshalYAML(node)
	return g
}

func makeGuardrailMap(key, val string) model.GuardrailRule {
	var g model.GuardrailRule
	node := &yaml.Node{
		Kind: yaml.MappingNode, Tag: "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
			{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"},
		},
	}
	_ = g.UnmarshalYAML(node)
	return g
}

// systematicDebugging: ~297 lines, 11 red flags, 8 rationalizations, 1 bash example, 4 phases
func skillSystematicDebugging() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "systematic-debugging",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"error_messages", "stack_traces", "git_diff", "environment_state"},
			Produces: []string{"root_cause", "failing_test", "targeted_fix"},
			Memory:   model.MemoryConversation,
		},
		Strategy: model.StrategyFacet{
			Approach: "sequential",
			Tools:    []string{"git", "bash", "grep", "read"},
			Steps: []string{
				"Root Cause Investigation: read error, trace origin, gather context",
				"Pattern Analysis: check recent commits, search for similar issues",
				"Hypothesis and Testing: form hypothesis, write failing test, verify",
				"Implementation: single targeted fix, run tests, verify no regression",
			},
		},
		Guardrails: []model.GuardrailRule{
			makeGuardrail("Never guess at root cause — investigate first"),
			makeGuardrail("Never make multiple changes at once"),
			makeGuardrail("Never skip writing a failing test before fixing"),
			makeGuardrail("Stop after 3 failed fix attempts — escalate to architecture review"),
			makeGuardrailMap("timeout", "30min"),
		},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelDetailed,
			Metrics:    []string{"fix_attempts", "time_to_root_cause"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadWrite,
			Network:    model.NetworkNone,
			Secrets:    []string{},
		},
		WhenToUse: model.WhenToUseFacet{
			Triggers:   []string{"Test failures", "Bug reports", "Unexpected behavior", "CI failures", "Runtime errors", "Regression detected"},
			DontUse:    []string{"Simple typo fixes", "Config-only changes", "Feature requests"},
			Especially: []string{"After 3+ failed fix attempts", "Under time pressure", "When root cause is unclear", "When multiple systems involved", "When previous fix introduced new bug"},
		},
		AntiPatterns: []model.AntiPattern{
			{Excuse: "I know what the problem is", Reality: "You have a hypothesis. Verify it."},
			{Excuse: "Let me just try this quick fix", Reality: "Quick fixes without root cause lead to whack-a-mole"},
			{Excuse: "The test passes locally", Reality: "Reproduce in the same environment that failed"},
			{Excuse: "This can't be related", Reality: "Follow the evidence, not assumptions"},
			{Excuse: "Let me revert and start over", Reality: "Understand why it failed first"},
			{Excuse: "It's probably a flaky test", Reality: "Verify flakiness before dismissing"},
			{Excuse: "Let me add a retry", Reality: "Retries hide bugs. Fix the root cause."},
			{Excuse: "Works on my machine", Reality: "The failing environment is the source of truth"},
		},
		Examples: []model.CodeExample{
			{Label: "Multi-layer diagnostic instrumentation", Code: "echo \"=== Environment ===\"\nenv | grep -i identity\necho \"=== Keychain ===\"\nsecurity find-identity -v -p codesigning\necho \"=== Build ===\"\nxcodebuild -showBuildSettings | grep SIGN", Lang: "bash"},
		},
	}
}

// testDrivenDevelopment: ~372 lines, 13 red flags, 11+3+4 anti-patterns, 6 code examples
func skillTDD() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "test-driven-development",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"requirements", "bug_reports", "source_code"},
			Produces: []string{"failing_test", "minimal_implementation", "refactored_code"},
			Memory:   model.MemoryConversation,
		},
		Strategy: model.StrategyFacet{
			Approach: "cyclical",
			Tools:    []string{"test_runner", "bash", "read", "write"},
			Steps: []string{
				"RED: Write a failing test that describes the desired behavior",
				"Verify: Run test, confirm it fails with expected error",
				"GREEN: Write minimal code to make the test pass",
				"Verify: Run test, confirm it passes",
				"REFACTOR: Clean up while keeping tests green",
				"Verify: Run all tests, confirm nothing broke",
			},
		},
		Guardrails: []model.GuardrailRule{
			makeGuardrail("Never write implementation before a failing test"),
			makeGuardrail("Never skip the RED verification step"),
			makeGuardrail("Never write more code than needed to pass the test"),
			makeGuardrail("Never refactor with failing tests"),
			makeGuardrail("Test must fail for the right reason"),
			makeGuardrailMap("timeout", "15min_per_cycle"),
		},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelStandard,
			Metrics:    []string{"red_green_cycles", "test_count", "coverage"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadWrite,
			Network:    model.NetworkNone,
			Secrets:    []string{},
		},
		WhenToUse: model.WhenToUseFacet{
			Triggers:   []string{"New features", "Bug fixes", "Refactoring", "Behavior changes"},
			DontUse:    []string{"Throwaway prototypes", "Generated code", "Configuration files"},
			Especially: []string{"When requirements are clear", "When correctness matters"},
		},
		AntiPatterns: []model.AntiPattern{
			{Excuse: "I'll add tests later", Reality: "Later never comes. Tests first."},
			{Excuse: "This is too simple to test", Reality: "Simple code changes. Tests catch regressions."},
			{Excuse: "Testing slows me down", Reality: "Debugging without tests is slower."},
			{Excuse: "I need to see the implementation first", Reality: "Tests ARE the spec. Write them first."},
			{Excuse: "100% coverage is overkill", Reality: "Test behavior, not lines. Coverage follows."},
			{Excuse: "Mocks are too complex", Reality: "Complex mocks = complex coupling. Simplify."},
		},
		Examples: []model.CodeExample{
			{Label: "Good: test describes behavior", Code: "describe('User.create', () => {\n  it('rejects duplicate emails', async () => {\n    await User.create({email: 'a@b.com'});\n    await expect(User.create({email: 'a@b.com'})).rejects.toThrow('duplicate');\n  });\n});", Lang: "typescript"},
			{Label: "Bad: test describes implementation", Code: "it('calls database.insert', () => {\n  // Testing HOW, not WHAT\n});", Lang: "typescript"},
			{Label: "Bug fix RED-GREEN cycle", Code: "# RED: reproduce the bug\nnpm test -- --grep 'handles null input'\n# Expected: FAIL\n\n# GREEN: fix the bug\nnpm test -- --grep 'handles null input'\n# Expected: PASS", Lang: "bash"},
		},
	}
}

// verificationBeforeCompletion: ~140 lines, 8 red flags, 7+8 rationalizations, 5 examples
func skillVerification() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "verification-before-completion",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"command_output", "exit_codes", "test_results", "vcs_diff"},
			Produces: []string{"evidence_backed_status", "verified_completion"},
			Memory:   model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Approach: "gate-checkpoint",
			Tools:    []string{"bash", "test_runner", "linter", "build"},
			Steps: []string{
				"IDENTIFY what needs verification for this claim",
				"RUN the verification command",
				"READ the full output (don't assume)",
				"VERIFY output matches expected state",
				"Only then claim completion with evidence",
			},
		},
		Guardrails: []model.GuardrailRule{
			makeGuardrail("Never claim tests pass without running them"),
			makeGuardrail("Never claim a fix works without verification output"),
			makeGuardrail("Never assume success from partial output"),
			makeGuardrail("Read EVERY line of test output"),
		},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelStandard,
			Metrics:    []string{"false_claims_caught"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadOnly,
			Network:    model.NetworkNone,
			Secrets:    []string{},
		},
		WhenToUse: model.WhenToUseFacet{
			Triggers:   []string{"Before claiming done", "Before committing", "Before creating PR", "Before reporting success", "Before delegating follow-up", "Before moving to next task"},
			Especially: []string{"When tests are involved", "When build must succeed"},
		},
		AntiPatterns: []model.AntiPattern{
			{Excuse: "Tests passed last time", Reality: "Run them again. State may have changed."},
			{Excuse: "I'm pretty sure it works", Reality: "Pretty sure != verified. Run the command."},
			{Excuse: "The change is trivial", Reality: "Trivial changes cause subtle regressions."},
			{Excuse: "I'll check after the commit", Reality: "Verify BEFORE the commit."},
			{Excuse: "The CI will catch it", Reality: "Don't push known-broken code to CI."},
		},
		Examples: []model.CodeExample{
			{Label: "Correct: verify then claim", Code: "go test ./...\n# Output: ok (all packages)\n# NOW claim: \"All tests pass\"", Lang: "bash"},
			{Label: "Wrong: claim without evidence", Code: "# Don't do this:\n# \"Tests should pass since I only changed a comment\"", Lang: "bash"},
		},
	}
}

// finishingBranch: ~201 lines, 4+4 red flags, 4 mistakes, 5 bash examples
func skillFinishingBranch() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "finishing-a-development-branch",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"test_results", "git_state", "user_choice"},
			Produces: []string{"merged_branch", "pull_request"},
			Memory:   model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Approach: "sequential-branching",
			Tools:    []string{"git", "bash", "gh", "test_runner"},
			Steps: []string{
				"Verify all tests pass",
				"Determine base branch",
				"Present 4 options to user",
				"Execute chosen option (merge/PR/keep/discard)",
				"Cleanup worktree if applicable",
			},
		},
		Guardrails: []model.GuardrailRule{
			makeGuardrail("Never proceed with failing tests"),
			makeGuardrail("Never force-push without explicit request"),
			makeGuardrail("Never delete work without typed confirmation"),
			makeGuardrail("Never merge without verifying tests on result"),
		},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelMinimal,
			Metrics:    []string{"completion_path_chosen"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadWrite,
			Network:    model.NetworkFull,
			Secrets:    []string{},
		},
		WhenToUse: model.WhenToUseFacet{
			Triggers: []string{"Implementation complete", "All tests pass", "Ready to integrate"},
			DontUse:  []string{"Work still in progress", "Tests failing"},
		},
		AntiPatterns: []model.AntiPattern{
			{Excuse: "Tests passed earlier", Reality: "Verify again before merge."},
			{Excuse: "I'll clean up the worktree later", Reality: "Clean up now or it accumulates."},
			{Excuse: "Just force-push it", Reality: "Force-push destroys history. Ask first."},
			{Excuse: "No need to confirm deletion", Reality: "Always require typed confirmation for destructive ops."},
		},
		Examples: []model.CodeExample{
			{Label: "Create PR with structured body", Code: "gh pr create --title \"feat: add X\" --body \"$(cat <<'EOF'\n## Summary\n- Added X\n## Test plan\n- [ ] Unit tests pass\nEOF\n)\"", Lang: "bash"},
			{Label: "Safe merge workflow", Code: "git checkout main && git pull && git merge feature-branch\ngo test ./...\ngit branch -d feature-branch", Lang: "bash"},
		},
	}
}

// requestingCodeReview: ~106 lines, 4 red flags, no anti-patterns, 2 examples
func skillCodeReview() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "requesting-code-review",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"git_shas", "implementation_description"},
			Produces: []string{"review_feedback", "severity_ranked_issues"},
			Memory:   model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Approach: "delegation",
			Tools:    []string{"git", "Task"},
			Steps: []string{
				"Get git SHAs (base and head)",
				"Dispatch code-reviewer subagent with template",
				"Triage feedback by severity (Critical > Important > Minor)",
			},
		},
		Guardrails: []model.GuardrailRule{
			makeGuardrail("Never skip review for completed tasks"),
			makeGuardrail("Never dismiss reviewer feedback without investigation"),
			makeGuardrail("If reviewer is wrong, verify with evidence before pushing back"),
		},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelMinimal,
			Metrics:    []string{"issues_found"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadOnly,
			Network:    model.NetworkNone,
			Secrets:    []string{},
		},
		WhenToUse: model.WhenToUseFacet{
			Triggers: []string{"Task implementation complete", "Major feature done", "Before merge"},
		},
		// No anti-patterns — this is the simplest superpowers skill
		Examples: []model.CodeExample{
			{Label: "Get SHAs for review", Code: "BASE=$(git merge-base HEAD main)\nHEAD=$(git rev-parse HEAD)\necho \"Review range: $BASE..$HEAD\"", Lang: "bash"},
		},
	}
}

// minimalForgenSkill: what forgent generates today with an empty template
func skillMinimalForgent() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "my-skill",
		Version: "0.1.0",
		Context: model.ContextFacet{
			Memory: model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Approach: "sequential",
		},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelMinimal,
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessNone,
			Network:    model.NetworkNone,
		},
	}
}

// --- Calibration: Vercel skills (guidelines/linter archetype) ---

// reactBestPractices: 58 rules, 8 categories, CRITICAL-LOW priority, code examples per rule
// SKILL.md=6KB index, AGENTS.md=83KB compiled. Very structured but no guardrails/anti-patterns.
func skillReactBestPractices() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "vercel-react-best-practices",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"source_code", "react_components", "next_config"},
			Produces: []string{"optimized_code", "performance_improvements"},
			Memory:   model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Approach: "rule-based",
			Tools:    []string{"read", "write", "grep"},
			Steps: []string{
				"Identify applicable rule category by priority (CRITICAL first)",
				"Check code against rules in that category",
				"Apply transformations following correct code examples",
				"Verify no regressions introduced",
			},
		},
		// No guardrails in Vercel skills — they're guidelines, not constraints
		Guardrails: nil,
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelMinimal,
			Metrics:    []string{},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadWrite,
			Network:    model.NetworkNone,
			Secrets:    []string{},
		},
		WhenToUse: model.WhenToUseFacet{
			Triggers: []string{
				"Writing new React components or Next.js pages",
				"Reviewing code for performance issues",
				"Refactoring existing React/Next.js code",
				"Optimizing bundle size or load times",
				"Implementing data fetching",
			},
		},
		// No anti-patterns — Vercel skills use bad/good code pairs instead
		Examples: []model.CodeExample{
			{Label: "Bad: sequential fetches (waterfall)", Code: "const user = await getUser(id);\nconst posts = await getPosts(user.id);\n// Sequential — posts waits for user", Lang: "typescript"},
			{Label: "Good: parallel fetches", Code: "const [user, posts] = await Promise.all([\n  getUser(id),\n  getPosts(id),\n]);", Lang: "typescript"},
		},
	}
}

// compositionPatterns: 10 rules, 4 categories, focused on React composition
func skillCompositionPatterns() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "vercel-composition-patterns",
		Version: "1.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"source_code", "component_architecture"},
			Produces: []string{"refactored_components"},
			Memory:   model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Approach: "rule-based",
			Tools:    []string{"read", "write"},
			Steps: []string{
				"Identify boolean prop proliferation or tight coupling",
				"Select applicable composition pattern",
				"Refactor using compound components or composition",
			},
		},
		Guardrails: nil,
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelMinimal,
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadWrite,
			Network:    model.NetworkNone,
		},
		WhenToUse: model.WhenToUseFacet{
			Triggers: []string{
				"Refactoring components with many boolean props",
				"Building reusable component libraries",
				"Designing flexible component APIs",
			},
		},
		Examples: []model.CodeExample{
			{Label: "Bad: boolean props", Code: "<Button primary large disabled loading />", Lang: "tsx"},
			{Label: "Good: composition", Code: "<Button variant=\"primary\" size=\"large\">\n  <Button.Spinner />\n  <Button.Label>Submit</Button.Label>\n</Button>", Lang: "tsx"},
		},
	}
}

// deployToVercel: operational skill, 12KB, very detailed step-by-step with bash commands
func skillDeployToVercel() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "deploy-to-vercel",
		Version: "3.0.0",
		Context: model.ContextFacet{
			Consumes: []string{"project_directory", "git_state", "vercel_config"},
			Produces: []string{"deployment_url", "preview_url"},
			Memory:   model.MemoryShortTerm,
		},
		Strategy: model.StrategyFacet{
			Approach: "sequential-branching",
			Tools:    []string{"bash", "vercel_cli", "git"},
			Steps: []string{
				"Gather project state (git remote, vercel link, CLI auth, teams)",
				"Choose deploy method based on state",
				"Execute deployment (git push, CLI deploy, or no-auth fallback)",
				"Return deployment URL to user",
			},
		},
		Guardrails: []model.GuardrailRule{
			makeGuardrail("Always deploy as preview unless user explicitly asks for production"),
			makeGuardrail("Never push without explicit user approval"),
			makeGuardrail("Do not curl or fetch the deployed URL to verify"),
		},
		Observability: model.ObservabilityFacet{
			TraceLevel: model.TraceLevelMinimal,
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadWrite,
			Network:    model.NetworkFull,
			Secrets:    []string{"VERCEL_TOKEN"},
		},
		WhenToUse: model.WhenToUseFacet{
			Triggers: []string{"Deploy my app", "Push this live", "Create a preview deployment"},
		},
		Examples: []model.CodeExample{
			{Label: "Deploy with CLI", Code: "vercel deploy . -y --no-wait --scope my-team\nvercel inspect <url>", Lang: "bash"},
			{Label: "Git push deploy", Code: "git add . && git commit -m \"deploy: update\" && git push", Lang: "bash"},
			{Label: "No-auth fallback", Code: "bash ~/.claude/skills/deploy-to-vercel/resources/deploy.sh .", Lang: "bash"},
		},
	}
}

func TestCalibration_ScoreRanges(t *testing.T) {
	type calibrationCase struct {
		name          string
		skill         model.SkillBehavior
		expectedRange [2]int // [min, max]
	}

	cases := []calibrationCase{
		// Gold standard: rich superpowers skills → expected 75-100
		{"systematic-debugging", skillSystematicDebugging(), [2]int{75, 100}},
		{"test-driven-development", skillTDD(), [2]int{75, 100}},
		{"verification-before-completion", skillVerification(), [2]int{70, 100}},
		{"finishing-a-development-branch", skillFinishingBranch(), [2]int{65, 100}},
		// Good but simpler skill → expected 55-85
		{"requesting-code-review", skillCodeReview(), [2]int{55, 85}},
		// Vercel skills: guidelines/linter archetype (no guardrails, no anti-patterns)
		{"vercel-react-best-practices", skillReactBestPractices(), [2]int{40, 70}},
		{"vercel-composition-patterns", skillCompositionPatterns(), [2]int{35, 65}},
		// Vercel operational skill: has guardrails, steps, examples
		{"vercel-deploy-to-vercel", skillDeployToVercel(), [2]int{50, 80}},
		// Minimal forgent template → expected 5-30
		{"minimal-forgent", skillMinimalForgent(), [2]int{5, 30}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ScoreSkill(tc.skill)
			t.Logf("%-35s → %3d/100  (ctx:%d strat:%d guard:%d obs:%d sec:%d wtu:%d ap:%d ex:%d)",
				tc.name, result.Total,
				result.Breakdown.Context, result.Breakdown.Strategy,
				result.Breakdown.Guardrails, result.Breakdown.Observability,
				result.Breakdown.Security, result.Breakdown.WhenToUse,
				result.Breakdown.AntiPatterns, result.Breakdown.Examples)
			assert.GreaterOrEqual(t, result.Total, tc.expectedRange[0],
				"Score %d below expected min %d", result.Total, tc.expectedRange[0])
			assert.LessOrEqual(t, result.Total, tc.expectedRange[1],
				"Score %d above expected max %d", result.Total, tc.expectedRange[1])
		})
	}
}

func TestCalibration_Ordering(t *testing.T) {
	rich := ScoreSkill(skillSystematicDebugging())
	medium := ScoreSkill(skillCodeReview())
	vercelOps := ScoreSkill(skillDeployToVercel())
	vercelGuide := ScoreSkill(skillReactBestPractices())
	minimal := ScoreSkill(skillMinimalForgent())

	t.Logf("Rich(systematic-debugging)=%d > Medium(code-review)=%d > VercelOps(deploy)=%d > VercelGuide(react)=%d > Minimal=%d",
		rich.Total, medium.Total, vercelOps.Total, vercelGuide.Total, minimal.Total)

	assert.Greater(t, rich.Total, medium.Total, "Rich workflow > simpler workflow")
	assert.Greater(t, medium.Total, vercelGuide.Total, "Workflow with anti-patterns > guidelines without")
	assert.Greater(t, vercelGuide.Total, minimal.Total, "Guidelines > empty template")
}
