package static

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/stretchr/testify/assert"
)

func TestPadding_DetectsFillerPhrases(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("Make sure to run tests.\nIt is important to validate.\nPlease note that this matters."),
	}
	check := &paddingCheck{}
	findings := check.Run(analysis)
	assert.Len(t, findings, 3)
	assert.Contains(t, findings[0].Issue, "make sure to")
}

func TestPadding_SkipsCodeBlocks(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("## Rules\nBe direct.\n```\nMake sure to initialize.\n```\nMore rules."),
	}
	check := &paddingCheck{}
	findings := check.Run(analysis)
	assert.Empty(t, findings, "filler inside code blocks should be skipped")
}

func TestPadding_NoFiller(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("Always run tests.\nNever skip linting.\nUse gofmt."),
	}
	check := &paddingCheck{}
	assert.Empty(t, check.Run(analysis))
}

func TestPadding_NilAnalysis(t *testing.T) {
	check := &paddingCheck{}
	assert.Nil(t, check.Run(nil))
}
