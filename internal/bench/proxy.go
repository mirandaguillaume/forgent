package bench

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/enricher"
	"github.com/mirandaguillaume/forgent/internal/scanner"
)

// sourceExts mirrors the scanner's source extensions for consistent counting.
var sourceExts = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".py": true, ".rs": true, ".java": true, ".rb": true, ".cs": true,
	".cpp": true, ".c": true, ".swift": true, ".kt": true,
	".php": true, ".scala": true, ".ex": true, ".exs": true,
}

// skipDirs mirrors the scanner's skip list.
var skipDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true, "__pycache__": true,
	".next": true, "dist": true, "build": true, ".claude": true,
	".github": true, "public": true, ".venv": true, "venv": true,
	"env": true, ".tox": true, "coverage": true, ".nyc_output": true,
	"target": true,
}

// RunProxy runs the proxy reachability benchmark.
// It scans the codebase, samples N random source files, and checks what
// percentage of them are reachable from the generated index.
func RunProxy(root string, samples int, seed int64) (*ProxyResult, error) {
	ctx, err := scanner.ScanCodebase(root)
	if err != nil {
		return nil, err
	}

	// Collect all source files.
	allFiles, err := collectSourceFiles(root)
	if err != nil {
		return nil, err
	}

	// Sample.
	sampled := sampleFiles(allFiles, samples, seed)

	// Check reachability.
	reachable := 0
	for _, f := range sampled {
		if isReachable(f, ctx.Structure) {
			reachable++
		}
	}

	rendered := enricher.RenderIndex(ctx)

	return &ProxyResult{
		TotalSourceFiles: len(allFiles),
		SampledFiles:     len(sampled),
		ReachableFiles:   reachable,
		Reachability:     safePercent(reachable, len(sampled)),
		IndexEntries:     len(ctx.Structure),
		IndexBytes:       len(rendered),
	}, nil
}

// isReachable checks if a source file's directory is covered by any index entry.
// A file at "apps/bo/common/hooks/useAuth.ts" is reachable if:
//   - An entry has Path == "apps/bo/common/hooks" (exact match)
//   - An entry has Path that is a prefix of the file's dir (e.g. "apps/bo/common")
//   - The file's dir is a prefix of an entry's Path (e.g. entry "apps/bo/common/hooks/deep")
func isReachable(filePath string, structure []scanner.DirEntry) bool {
	dir := filepath.Dir(filePath)
	for _, e := range structure {
		if e.Path == dir {
			return true
		}
		if strings.HasPrefix(dir, e.Path+"/") {
			return true
		}
		if strings.HasPrefix(e.Path, dir+"/") {
			return true
		}
	}
	return false
}

// collectSourceFiles walks the filesystem and returns all source file paths
// relative to root.
func collectSourceFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		if sourceExts[ext] {
			rel, _ := filepath.Rel(root, path)
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

// sampleFiles returns up to n random files from the list using the given seed.
func sampleFiles(files []string, n int, seed int64) []string {
	if len(files) <= n {
		return files
	}
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
	return files[:n]
}

func safePercent(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den) * 100
}
