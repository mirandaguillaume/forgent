package bench

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// MartianPR represents one PR from the Martian golden comment dataset.
type MartianPR struct {
	PRTitle  string           `json:"pr_title"`
	URL      string           `json:"url"`
	Comments []MartianComment `json:"comments"`
}

// MartianComment is a single golden comment with severity.
type MartianComment struct {
	Comment  string `json:"comment"`
	Severity string `json:"severity"`
}

// MartianTasks loads all golden comment JSONs from dir and converts to H2HTask slice.
// Each PR becomes one H2HTask; each comment becomes one criterion.
func MartianTasks(dir string) ([]H2HTask, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read martian dir: %w", err)
	}

	var tasks []H2HTask
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		category := strings.TrimSuffix(entry.Name(), ".json")

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		var prs []MartianPR
		if err := json.Unmarshal(data, &prs); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		for _, pr := range prs {
			hash := sha256.Sum256([]byte(pr.URL))
			id := fmt.Sprintf("%s-%x", category, hash)
			if len(id) > 20 {
				id = id[:20]
			}
			var criteria []string
			var severities []string
			for _, c := range pr.Comments {
				criteria = append(criteria, c.Comment)
				severities = append(severities, c.Severity)
			}
			tasks = append(tasks, H2HTask{
				ID:         id,
				Category:   category,
				Title:      pr.PRTitle,
				URL:        pr.URL,
				Criteria:   criteria,
				Severities: severities,
			})
		}
	}

	return tasks, nil
}

// ParsePRURL extracts owner, repo, and PR number from a GitHub PR URL.
func ParsePRURL(url string) (owner, repo string, number int, err error) {
	url = strings.TrimSuffix(url, "/")
	parts := strings.Split(url, "/")
	if len(parts) < 5 || parts[len(parts)-2] != "pull" {
		return "", "", 0, fmt.Errorf("invalid PR URL: %s", url)
	}
	number, err = strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number in %s: %w", url, err)
	}
	owner = parts[len(parts)-4]
	repo = parts[len(parts)-3]
	return owner, repo, number, nil
}

var diffCache = struct {
	mu sync.Mutex
	m  map[string]string
}{m: make(map[string]string)}

// FetchPRDiff fetches the diff for a GitHub PR using the gh CLI.
// Results are cached in memory for the lifetime of the process.
func FetchPRDiff(url string) (string, error) {
	diffCache.mu.Lock()
	if cached, ok := diffCache.m[url]; ok {
		diffCache.mu.Unlock()
		return cached, nil
	}
	diffCache.mu.Unlock()

	owner, repo, number, err := ParsePRURL(url)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("gh", "pr", "diff", strconv.Itoa(number),
		"--repo", fmt.Sprintf("%s/%s", owner, repo))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh pr diff %s: %w", url, err)
	}

	diff := string(out)

	diffCache.mu.Lock()
	diffCache.m[url] = diff
	diffCache.mu.Unlock()

	return diff, nil
}
