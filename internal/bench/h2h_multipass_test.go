package bench

import (
	"fmt"
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericPassStrategy_Pass1(t *testing.T) {
	s := &GenericPassStrategy{}
	prompt := s.ReviewPrompt(1, "You review code.", "func foo() {}", nil)
	assert.Contains(t, prompt, "You review code.")
	assert.Contains(t, prompt, "func foo() {}")
	assert.NotContains(t, prompt, "re-examine")
}

func TestGenericPassStrategy_Pass2_IncludesHistory(t *testing.T) {
	s := &GenericPassStrategy{}
	history := []PassRecord{
		{Pass: 1, Response: "Found a null deref on line 5."},
	}
	prompt := s.ReviewPrompt(2, "You review code.", "func foo() {}", history)
	assert.Contains(t, prompt, "You review code.")
	assert.Contains(t, prompt, "func foo() {}")
	assert.Contains(t, prompt, "Found a null deref on line 5.")
	assert.Contains(t, prompt, "Review Pass 1")
	assert.Contains(t, prompt, "re-examine")
	assert.Contains(t, prompt, "Pass 2 instructions")
}

func TestGenericPassStrategy_Pass3_AccumulatesHistory(t *testing.T) {
	s := &GenericPassStrategy{}
	history := []PassRecord{
		{Pass: 1, Response: "Issue A."},
		{Pass: 2, Response: "Issue B."},
	}
	prompt := s.ReviewPrompt(3, "Sys.", "code", history)
	assert.Contains(t, prompt, "Review Pass 1")
	assert.Contains(t, prompt, "Review Pass 2")
	assert.Contains(t, prompt, "Issue A.")
	assert.Contains(t, prompt, "Issue B.")
	assert.Contains(t, prompt, "Pass 3 instructions")
}

func TestStagedPassStrategy_Pass1_SameAsGeneric(t *testing.T) {
	s := &StagedPassStrategy{
		Stages: []model.Stage{{Name: "preflight", Strategy: "sequential", Skills: []string{"pr-triage"}}},
	}
	prompt := s.ReviewPrompt(1, "System.", "code here", nil)
	assert.Contains(t, prompt, "System.")
	assert.Contains(t, prompt, "code here")
	assert.NotContains(t, prompt, "preflight")
}

func TestStagedPassStrategy_Pass2_UsesStageGuidance(t *testing.T) {
	s := &StagedPassStrategy{
		Stages: []model.Stage{
			{Name: "preflight", Strategy: "sequential", Skills: []string{"pr-triage"}},
			{Name: "analysis", Strategy: "parallel", Skills: []string{"bug-scanner", "git-history-reviewer"}},
		},
		SkillSteps: map[string][]string{
			"bug-scanner":          {"scan for null derefs", "check off-by-one errors"},
			"git-history-reviewer": {"check blame context", "identify regressions"},
		},
	}
	history := []PassRecord{{Pass: 1, Response: "Initial findings."}}
	prompt := s.ReviewPrompt(2, "Sys.", "code", history)

	assert.Contains(t, prompt, "analysis")
	assert.Contains(t, prompt, "parallel")
	assert.Contains(t, prompt, "bug-scanner")
	assert.Contains(t, prompt, "scan for null derefs")
	assert.Contains(t, prompt, "git-history-reviewer")
	assert.Contains(t, prompt, "check blame context")
	assert.Contains(t, prompt, "Initial findings.")
}

func TestStagedPassStrategy_PassExceedsStages_ClampToLast(t *testing.T) {
	s := &StagedPassStrategy{
		Stages: []model.Stage{
			{Name: "preflight", Strategy: "sequential", Skills: []string{"pr-triage"}},
			{Name: "publish", Strategy: "sequential", Skills: []string{"pr-commenter"}},
		},
		SkillSteps: map[string][]string{
			"pr-commenter": {"write inline comments"},
		},
	}
	history := []PassRecord{
		{Pass: 1, Response: "A"},
		{Pass: 2, Response: "B"},
		{Pass: 3, Response: "C"},
	}
	// Pass 4 exceeds 2 stages — should clamp to last stage (publish)
	prompt := s.ReviewPrompt(4, "Sys.", "code", history)
	assert.Contains(t, prompt, "publish")
	assert.Contains(t, prompt, "write inline comments")
}

func TestResolvePassStrategy_NonStaged(t *testing.T) {
	c := H2HContestant{Name: "baseline"}
	s := ResolvePassStrategy(c)
	_, ok := s.(*GenericPassStrategy)
	assert.True(t, ok, "should return GenericPassStrategy for non-staged contestant")
}

func TestResolvePassStrategy_Staged(t *testing.T) {
	c := H2HContestant{
		Name: "forgent/staged-reviewer",
		Stages: []model.Stage{
			{Name: "analysis", Skills: []string{"bug-scanner"}},
		},
		SkillSteps: map[string][]string{"bug-scanner": {"step1"}},
	}
	s := ResolvePassStrategy(c)
	staged, ok := s.(*StagedPassStrategy)
	assert.True(t, ok, "should return StagedPassStrategy for staged contestant")
	assert.Equal(t, 1, len(staged.Stages))
}

// mockProvider is a simple mock for testing multi-pass evaluation.
type mockProvider struct {
	calls     int
	responses []string
}

func (m *mockProvider) Complete(prompt string) (string, error) {
	idx := m.calls
	m.calls++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return "", fmt.Errorf("no more mock responses")
}

func TestEvalTaskMultiPass_SinglePass_DelegatesToEvalTask(t *testing.T) {
	provider := &mockProvider{
		responses: []string{
			"Found a bug on line 3.",                                                   // review
			`{"criteria_hits": [true, false], "reasoning": "caught one of two issues"}`, // judge
		},
	}
	task := H2HTask{
		ID:         "test-1",
		Code:       "func foo() {}",
		Criteria:   []string{"check null", "check overflow"},
		Severities: []string{"High", "Low"},
	}
	contestant := H2HContestant{Name: "test-agent", Prompt: "Review code.", Words: 2}

	result := evalTaskMultiPass(contestant, task, provider, 1)
	assert.Equal(t, 2, provider.calls, "single-pass should make 2 LLM calls (review + judge)")
	assert.Equal(t, "test-1", result.TaskID)
	assert.Greater(t, result.Score, 0.0)
}

func TestEvalTaskMultiPass_ThreePasses(t *testing.T) {
	provider := &mockProvider{
		responses: []string{
			"Pass 1: Found null deref.",                                            // review pass 1
			"Pass 2: Also found off-by-one.",                                       // review pass 2
			"Pass 3: Consolidated review with both issues.",                         // review pass 3
			`{"criteria_hits": [true, true], "reasoning": "both issues addressed"}`, // judge
		},
	}
	task := H2HTask{
		ID:         "test-2",
		Code:       "func bar() {}",
		Criteria:   []string{"check null", "check overflow"},
		Severities: []string{"Critical", "Medium"},
	}
	contestant := H2HContestant{Name: "multi-agent", Prompt: "You review code.", Words: 3}

	result := evalTaskMultiPass(contestant, task, provider, 3)
	assert.Equal(t, 4, provider.calls, "3 passes + 1 judge = 4 LLM calls")
	assert.Equal(t, 4, result.LLMCalls)
	assert.Equal(t, 3, result.Passes)
	assert.Equal(t, 3, len(result.PassDetails))
	assert.Equal(t, 100.0, result.Score, "both criteria hit should give 100%")
}

func TestEvalTaskMultiPass_ErrorOnPass2(t *testing.T) {
	provider := &mockProvider{
		responses: []string{
			"Pass 1 review.", // review pass 1
			// pass 2 will error (no more responses)
		},
	}
	task := H2HTask{
		ID:       "test-3",
		Code:     "code",
		Criteria: []string{"check"},
	}
	contestant := H2HContestant{Name: "err-agent", Prompt: "Sys.", Words: 1}

	result := evalTaskMultiPass(contestant, task, provider, 3)
	require.Contains(t, result.Reasoning, "error on pass 2")
	assert.Equal(t, 2, result.Passes, "should report pass where error occurred")
	assert.Equal(t, 1, len(result.PassDetails), "should have 1 successful pass record")
	assert.Equal(t, 0.0, result.Score, "errored task should have 0 score")
}
