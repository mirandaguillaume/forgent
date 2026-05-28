package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T6.12 — AC8: --baseline and --save-baseline flags must be registered on the bench subcommand.
func TestCLI_BenchAcceptsBaselineFlag(t *testing.T) {
	benchCmd, _, err := rootCmd.Find([]string{"bench"})
	require.NoError(t, err)
	require.NotNil(t, benchCmd, "bench subcommand must exist")

	baseline := benchCmd.Flags().Lookup("baseline")
	require.NotNil(t, baseline, "--baseline flag must be registered")
	assert.Equal(t, "string", baseline.Value.Type())
	assert.Equal(t, "", baseline.DefValue, "--baseline default must be empty")

	saveBaseline := benchCmd.Flags().Lookup("save-baseline")
	require.NotNil(t, saveBaseline, "--save-baseline flag must be registered")
	assert.Equal(t, "string", saveBaseline.Value.Type())
	assert.Equal(t, "", saveBaseline.DefValue, "--save-baseline default must be empty")
}
