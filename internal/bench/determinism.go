package bench

import (
	"bytes"
	"os"
	"path/filepath"
)

// DeterminismResult holds the outcome of the build determinism benchmark.
type DeterminismResult struct {
	Identical bool
	Runs      int
	DiffCount int
	DiffFiles []string
}

// RunDeterminism runs the build pipeline N times and compares outputs byte-for-byte.
func RunDeterminism(skillsDir, agentsDir, target string, runs int) (*DeterminismResult, error) {
	if runs < 2 {
		runs = 2
	}

	// Build into separate temp directories
	dirs := make([]string, runs)
	for i := 0; i < runs; i++ {
		tmpDir, err := os.MkdirTemp("", "bench-det-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmpDir)
		dirs[i] = tmpDir

		if err := buildToDir(skillsDir, agentsDir, tmpDir, target); err != nil {
			return nil, err
		}
	}

	// Compare all runs against the first
	result := &DeterminismResult{Runs: runs, Identical: true}
	refDir := dirs[0]

	err := filepath.Walk(refDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(refDir, path)
		refData, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Normalize: replace temp dir paths so comparisons ignore path differences
		normalizedRef := bytes.ReplaceAll(refData, []byte(refDir), []byte("<OUTPUT>"))

		for i := 1; i < runs; i++ {
			otherPath := filepath.Join(dirs[i], rel)
			otherData, err := os.ReadFile(otherPath)
			if err != nil {
				result.Identical = false
				result.DiffCount++
				result.DiffFiles = append(result.DiffFiles, rel+" (missing in run "+string(rune('0'+i+1))+")")
				continue
			}
			normalizedOther := bytes.ReplaceAll(otherData, []byte(dirs[i]), []byte("<OUTPUT>"))
			if !bytes.Equal(normalizedRef, normalizedOther) {
				result.Identical = false
				result.DiffCount++
				result.DiffFiles = append(result.DiffFiles, rel)
			}
		}
		return nil
	})

	return result, err
}
