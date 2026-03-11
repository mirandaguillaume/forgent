package main

import (
	"os"

	"github.com/mirandaguillaume/forgent/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
