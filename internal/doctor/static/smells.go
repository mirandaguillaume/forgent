package static

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
)

func init() {
	RegisterCheck(&smellsCheck{})
}

// smellCategory is used for all agent smell findings.
const smellCategory = "agent_smell"

type smellsCheck struct{}

func (s *smellsCheck) ID() string              { return "agent-smells" }
func (s *smellsCheck) Tags() []string          { return []string{"thorough"} }
func (s *smellsCheck) Category() string        { return smellCategory }
func (s *smellsCheck) DefaultSeverity() string { return "moderate" }

func (s *smellsCheck) Run(analysis *parser.AgentAnalysis) []llmutil.Finding {
	if analysis == nil || len(analysis.RawContent) == 0 {
		return nil
	}

	var findings []llmutil.Finding

	if f := detectGodAgent(analysis); f != nil {
		findings = append(findings, *f)
	}
	if f := detectCopyPaste(analysis); f != nil {
		findings = append(findings, *f)
	}
	if f := detectLLMGenerated(analysis); f != nil {
		findings = append(findings, *f)
	}
	if f := detectCheckboxAgent(analysis); f != nil {
		findings = append(findings, *f)
	}
	if f := detectOverConstrained(analysis); f != nil {
		findings = append(findings, *f)
	}

	return findings
}

// --- God Agent: >3000 words AND >8 sections ---

func detectGodAgent(analysis *parser.AgentAnalysis) *llmutil.Finding {
	words := len(strings.Fields(string(analysis.RawContent)))
	sections := countH2PlusSections(analysis)

	if words <= 3000 || sections <= 8 {
		return nil
	}

	return &llmutil.Finding{
		Category:             smellCategory,
		Issue:                fmt.Sprintf("God agent: %d words across %d sections", words, sections),
		Confidence:           "moderate",
		CitationID:           "decomposition",
		CurrentState:         fmt.Sprintf("%d words, %d sections (threshold: >3000 words AND >8 sections)", words, sections),
		SuggestedImprovement: "Decompose into 2-3 focused skills. Research shows 4+ skills = diminishing returns (SkillsBench, +5.2pp vs +20.0pp for 2-3)",
	}
}

func countH2PlusSections(analysis *parser.AgentAnalysis) int {
	count := 0
	for _, s := range analysis.Sections {
		if s.Level >= 2 {
			count++
		}
	}
	return count
}

// --- Copy-Paste Agent: >80% Jaccard with default templates ---

// Default template fingerprints — representative phrases that indicate a generic,
// uncustomized agent file. We check if the file is MOSTLY these phrases.
// The approach: count how many template phrases appear in the file. If >80% of
// template phrases are present and the file has few additional unique phrases,
// it's likely a copy-paste.
var defaultTemplatePhrases = map[string][]string{
	"claude_default": {
		"you are claude", "ai assistant", "made by anthropic",
		"helpful harmless and honest", "respond to the human",
		"helpful and informative", "follow instructions carefully",
		"think step by step", "provide accurate information",
		"if you are unsure", "let the user know",
	},
	"copilot_default": {
		"you are a helpful assistant", "helps developers write code",
		"follow best practices", "provide clear explanations",
		"write clean code", "add comments to explain",
		"use proper error handling", "follow coding conventions",
		"suggest improvements when appropriate",
	},
}

func detectCopyPaste(analysis *parser.AgentAnalysis) *llmutil.Finding {
	content := strings.ToLower(string(analysis.RawContent))
	if len(content) < 50 {
		return nil
	}

	for templateName, phrases := range defaultTemplatePhrases {
		matched := 0
		for _, phrase := range phrases {
			if strings.Contains(content, phrase) {
				matched++
			}
		}
		ratio := float64(matched) / float64(len(phrases))
		if ratio > 0.80 {
			return &llmutil.Finding{
				Category:             smellCategory,
				Issue:                fmt.Sprintf("Copy-paste agent: %.0f%% of %s template phrases found", ratio*100, templateName),
				Confidence:           "moderate",
				CurrentState:         fmt.Sprintf("%d/%d template phrases matched", matched, len(phrases)),
				SuggestedImprovement: "Customize this file for your project. Default templates provide no project-specific guidance.",
			}
		}
	}

	return nil
}

func toWordSet(text string) map[string]bool {
	words := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(text)) {
		clean := strings.Trim(w, ".,;:!?\"'()[]`")
		if len(clean) > 2 {
			words[clean] = true
		}
	}
	return words
}

func jaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	intersection := 0
	for w := range a {
		if b[w] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// --- LLM-Generated Agent: transition phrases + numbered lists ---

var llmTransitions = []string{
	"additionally", "furthermore", "moreover", "in addition",
	"it's worth noting", "it is worth noting", "it should be noted",
}

var numberedListRe = regexp.MustCompile(`^\s*\d+\.\s`)

func detectLLMGenerated(analysis *parser.AgentAnalysis) *llmutil.Finding {
	content := normalizeApostrophes(strings.ToLower(string(analysis.RawContent)))

	transitionCount := 0
	for _, t := range llmTransitions {
		transitionCount += strings.Count(content, t)
	}

	numberedCount := 0
	for _, line := range strings.Split(content, "\n") {
		if numberedListRe.MatchString(line) {
			numberedCount++
		}
	}

	if transitionCount <= 3 || numberedCount <= 5 {
		return nil
	}

	return &llmutil.Finding{
		Category:             smellCategory,
		Issue:                fmt.Sprintf("LLM-generated agent: %d transition phrases, %d numbered items", transitionCount, numberedCount),
		Confidence:           "low",
		CitationID:           "redundancy",
		CurrentState:         fmt.Sprintf("%d LLM transition phrases (Additionally, Furthermore...) and %d numbered list items", transitionCount, numberedCount),
		SuggestedImprovement: "LLM-generated context files have -3% impact (Gloaguen et al.). Rewrite with concise, human-authored instructions.",
	}
}

// --- Checkbox Agent: many sections, each too shallow ---

func detectCheckboxAgent(analysis *parser.AgentAnalysis) *llmutil.Finding {
	sections := analysis.Sections
	if len(sections) < 8 {
		return nil
	}

	totalWords := 0
	for _, s := range sections {
		totalWords += len(strings.Fields(s.Content))
	}

	avgWords := totalWords / len(sections)
	if avgWords >= 20 {
		return nil
	}

	return &llmutil.Finding{
		Category:             smellCategory,
		Issue:                fmt.Sprintf("Checkbox agent: %d sections with avg %d words each", len(sections), avgWords),
		Confidence:           "moderate",
		CurrentState:         fmt.Sprintf("%d sections present but averaging only %d words per section (threshold: <20)", len(sections), avgWords),
		SuggestedImprovement: "All sections present but too shallow. Add substance — specific instructions, examples, and criteria to each section.",
	}
}

// --- Over-Constrained Agent: prohibitions >> permissions ---

// normalizeApostrophes replaces typographic apostrophes with ASCII ones.
func normalizeApostrophes(s string) string {
	s = strings.ReplaceAll(s, "\u2019", "'") // right single quotation mark
	s = strings.ReplaceAll(s, "\u2018", "'") // left single quotation mark
	s = strings.ReplaceAll(s, "\u201C", "\"") // left double quotation mark
	s = strings.ReplaceAll(s, "\u201D", "\"") // right double quotation mark
	return s
}

// countWordMatches counts occurrences of target words/phrases in a word list.
// Multi-word phrases are checked against the original content via strings.Count.
// Single words are matched as whole words from the Fields-split list.
func countWordMatches(words []string, targets []string) int {
	count := 0
	for _, target := range targets {
		if strings.Contains(target, " ") {
			// Multi-word phrase: count in joined content (already lowercase)
			content := strings.Join(words, " ")
			count += strings.Count(content, target)
		} else {
			// Single word: match whole words only
			for _, w := range words {
				clean := strings.Trim(w, ".,;:!?\"'()[]`")
				if clean == target {
					count++
				}
			}
		}
	}
	return count
}

var prohibitionWords = []string{"never", "don't", "do not", "must not", "avoid", "prohibited", "forbidden"}
var permissionWords = []string{"should", "can", "allowed", "encouraged", "recommended", "free to"}

func detectOverConstrained(analysis *parser.AgentAnalysis) *llmutil.Finding {
	content := normalizeApostrophes(strings.ToLower(string(analysis.RawContent)))
	words := strings.Fields(content)

	prohibitions := countWordMatches(words, prohibitionWords)
	permissions := countWordMatches(words, permissionWords)

	if prohibitions <= 10 || permissions >= 3 {
		return nil
	}

	return &llmutil.Finding{
		Category:             smellCategory,
		Issue:                fmt.Sprintf("Over-constrained agent: %d prohibitions, %d permissions", prohibitions, permissions),
		Confidence:           "moderate",
		CurrentState:         fmt.Sprintf("%d prohibitions (never, don't, avoid...) but only %d positive permissions (should, can, allowed...)", prohibitions, permissions),
		SuggestedImprovement: "Add positive guidance about what to do, not just what not to do. Agents need both constraints and direction.",
	}
}
