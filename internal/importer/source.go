package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SourceType indicates where an import source comes from.
type SourceType int

const (
	SourceLocalFile SourceType = iota
	SourceLocalDir
	SourceVercel
)

// Framework identifies the agent framework a source belongs to.
type Framework int

const (
	FrameworkUnknown Framework = iota
	FrameworkClaude
	FrameworkCopilot
)

// Source represents a resolved import source with its content and framework.
type Source struct {
	Name      string
	Path      string
	Content   string
	Framework Framework
}

// DetectSourceType determines the type of import source from an input string.
// A "vercel:" prefix indicates a Vercel source; an existing directory indicates
// a local directory; everything else is treated as a local file.
func DetectSourceType(input string) SourceType {
	if strings.HasPrefix(input, "vercel:") {
		return SourceVercel
	}
	info, err := os.Stat(input)
	if err == nil && info.IsDir() {
		return SourceLocalDir
	}
	return SourceLocalFile
}

// DetectFramework guesses the agent framework from a file path.
func DetectFramework(path string) Framework {
	normalized := filepath.ToSlash(path)
	if strings.Contains(normalized, ".claude/") {
		return FrameworkClaude
	}
	if strings.Contains(normalized, ".github/") {
		return FrameworkCopilot
	}
	return FrameworkUnknown
}

// ResolveSources resolves the input string into a list of Source values.
// For a local file it reads the file content; for a directory it globs *.md
// files; for Vercel it returns an error indicating the feature is not yet
// implemented.
func ResolveSources(input string) ([]Source, error) {
	st := DetectSourceType(input)
	switch st {
	case SourceVercel:
		return nil, fmt.Errorf("vercel source not yet implemented")
	case SourceLocalDir:
		return resolveDir(input)
	default:
		return resolveFile(input)
	}
}

func resolveFile(path string) ([]Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading source file: %w", err)
	}
	return []Source{
		{
			Name:      filepath.Base(path),
			Path:      path,
			Content:   string(data),
			Framework: DetectFramework(path),
		},
	}, nil
}

func resolveDir(dir string) ([]Source, error) {
	pattern := filepath.Join(dir, "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("globbing directory: %w", err)
	}

	var sources []Source
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", m, err)
		}
		sources = append(sources, Source{
			Name:      filepath.Base(m),
			Path:      m,
			Content:   string(data),
			Framework: DetectFramework(m),
		})
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no .md files found in %s", dir)
	}
	return sources, nil
}
