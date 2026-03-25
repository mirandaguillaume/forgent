package cmd

import (
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "forgent-bench",
	Short:   "Forgent Bench — Benchmark agent composition quality",
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}
