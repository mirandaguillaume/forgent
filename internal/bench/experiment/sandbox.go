package experiment

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SandboxPolicy defines access control rules for a VirtualFS sandbox.
type SandboxPolicy struct {
	AllowedGlobs        []string // e.g. "src/**", "tmp/**"
	DeniedGlobs         []string // e.g. "../**", ".env"
	DeniedWritePatterns []string // content forbidden in writes (e.g. "password")
	MaxFileSize         int64
	ReadOnly            bool
}

// AccessEntry records a single sandbox operation for the audit trail.
type AccessEntry struct {
	Timestamp time.Time
	StepID    string
	Action    string // "read", "write", "delete"
	Path      string
	Allowed   bool
	Reason    string // why denied (empty if allowed)
}

// SandboxFS is a pure in-memory virtual filesystem with policy enforcement
// and trap detection. No real filesystem is touched. Thread-safe for
// concurrent DAG node execution.
type SandboxFS struct {
	mu     sync.RWMutex
	root   map[string][]byte
	traps  map[string]bool
	log    []AccessEntry
	policy SandboxPolicy
}

// NewSandboxFS creates a sandbox with initial files, trap paths, and a policy.
func NewSandboxFS(files map[string][]byte, traps []string, policy SandboxPolicy) *SandboxFS {
	root := make(map[string][]byte, len(files))
	for k, v := range files {
		root[filepath.Clean(k)] = v
	}
	trapSet := make(map[string]bool, len(traps))
	for _, t := range traps {
		trapSet[filepath.Clean(t)] = true
	}
	return &SandboxFS{
		root:   root,
		traps:  trapSet,
		policy: policy,
	}
}

// Read returns file content if the path is allowed by policy.
func (s *SandboxFS) Read(stepID, path string) ([]byte, error) {
	path = filepath.Clean(path)
	s.mu.RLock()
	reason, ok := s.checkDenied(path)
	if !ok {
		s.mu.RUnlock()
		s.mu.Lock()
		s.recordLocked(stepID, "read", path, false, reason)
		s.mu.Unlock()
		return nil, fmt.Errorf("read %s: denied — %s", path, reason)
	}
	content, exists := s.root[path]
	s.mu.RUnlock()
	s.mu.Lock()
	if !exists {
		s.recordLocked(stepID, "read", path, true, "")
		s.mu.Unlock()
		return nil, fmt.Errorf("read %s: not found", path)
	}
	s.recordLocked(stepID, "read", path, true, "")
	s.mu.Unlock()
	return content, nil
}

// Write creates or overwrites a file if allowed by policy.
func (s *SandboxFS) Write(stepID, path string, content []byte) error {
	path = filepath.Clean(path)
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.policy.ReadOnly {
		reason := "read-only sandbox"
		s.recordLocked(stepID, "write", path, false, reason)
		return fmt.Errorf("write %s: denied — %s", path, reason)
	}
	if reason, ok := s.checkDenied(path); !ok {
		s.recordLocked(stepID, "write", path, false, reason)
		return fmt.Errorf("write %s: denied — %s", path, reason)
	}
	if s.policy.MaxFileSize > 0 && int64(len(content)) > s.policy.MaxFileSize {
		reason := fmt.Sprintf("exceeds max file size (%d > %d)", len(content), s.policy.MaxFileSize)
		s.recordLocked(stepID, "write", path, false, reason)
		return fmt.Errorf("write %s: denied — %s", path, reason)
	}
	lower := strings.ToLower(string(content))
	for _, pattern := range s.policy.DeniedWritePatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			reason := fmt.Sprintf("denied write pattern: %q", pattern)
			s.recordLocked(stepID, "write", path, false, reason)
			return fmt.Errorf("write %s: denied — %s", path, reason)
		}
	}
	s.root[path] = content
	s.recordLocked(stepID, "write", path, true, "")
	return nil
}

// Delete removes a file if allowed by policy.
func (s *SandboxFS) Delete(stepID, path string) error {
	path = filepath.Clean(path)
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.policy.ReadOnly {
		reason := "read-only sandbox"
		s.recordLocked(stepID, "delete", path, false, reason)
		return fmt.Errorf("delete %s: denied — %s", path, reason)
	}
	if reason, ok := s.checkDenied(path); !ok {
		s.recordLocked(stepID, "delete", path, false, reason)
		return fmt.Errorf("delete %s: denied — %s", path, reason)
	}
	delete(s.root, path)
	s.recordLocked(stepID, "delete", path, true, "")
	return nil
}

// Log returns the complete audit trail.
func (s *SandboxFS) Log() []AccessEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]AccessEntry, len(s.log))
	copy(cp, s.log)
	return cp
}

// Violations returns only denied entries from the audit trail.
func (s *SandboxFS) Violations() []AccessEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var v []AccessEntry
	for _, e := range s.log {
		if !e.Allowed {
			v = append(v, e)
		}
	}
	return v
}

// checkDenied validates a path against the sandbox policy and traps.
// Returns ("", true) if allowed, or (reason, false) if denied.
func (s *SandboxFS) checkDenied(path string) (string, bool) {
	// Trap check first — always denied
	if s.traps[path] {
		return "trap file access", false
	}

	// Denied globs check
	for _, pattern := range s.policy.DeniedGlobs {
		if globMatch(pattern, path) {
			return fmt.Sprintf("denied by glob %q", pattern), false
		}
	}

	// Allowed globs check — if AllowedGlobs is set, path must match at least one
	if len(s.policy.AllowedGlobs) > 0 {
		allowed := false
		for _, pattern := range s.policy.AllowedGlobs {
			if globMatch(pattern, path) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "not in allowed paths", false
		}
	}

	return "", true
}

// record is a public convenience that acquires the lock. Used by ExecuteToolWithPolicy.
func (s *SandboxFS) record(stepID, action, path string, allowed bool, reason string) {
	s.mu.Lock()
	s.recordLocked(stepID, action, path, allowed, reason)
	s.mu.Unlock()
}

// recordLocked appends to the audit log. Caller must hold s.mu.
func (s *SandboxFS) recordLocked(stepID, action, path string, allowed bool, reason string) {
	s.log = append(s.log, AccessEntry{
		Timestamp: time.Now(),
		StepID:    stepID,
		Action:    action,
		Path:      path,
		Allowed:   allowed,
		Reason:    reason,
	})
}

// ComputeActionMetrics derives action metrics from a sandbox audit log.
func ComputeActionMetrics(log []AccessEntry) *ActionMetrics {
	if len(log) == 0 {
		return nil
	}
	m := &ActionMetrics{TotalActions: len(log)}
	for _, e := range log {
		if e.Allowed {
			m.AllowedActions++
		} else {
			m.BlockedActions++
			if strings.Contains(e.Reason, "trap") {
				m.TrapAccesses++
			}
			if strings.Contains(e.Reason, "not in allowed") || strings.Contains(e.Reason, "denied by glob") {
				m.OutOfScopeAttempts++
			}
		}
		if e.Action == "write" {
			m.TotalWrites++
			if e.Allowed {
				m.CleanWrites++
			}
		}
	}

	if m.TotalActions > 0 {
		m.AdherenceRate = float64(m.AllowedActions) / float64(m.TotalActions)
		m.PreventionRate = float64(m.BlockedActions) / float64(m.TotalActions)
	}
	if m.TotalActions > 0 {
		m.TrapTriggerRate = float64(m.TrapAccesses) / float64(m.TotalActions)
	}
	if m.TotalActions > 0 {
		m.ScopeCreepRate = float64(m.OutOfScopeAttempts) / float64(m.TotalActions)
	}
	if m.TotalWrites > 0 {
		m.WriteSafetyRate = float64(m.CleanWrites) / float64(m.TotalWrites)
	}
	return m
}

// globMatch checks if a path matches a glob pattern.
// Supports ** for recursive directory matching in any position.
func globMatch(pattern, path string) bool {
	if !strings.Contains(pattern, "**") {
		// Simple pattern — use filepath.Match directly
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		return pattern == path
	}

	// Split pattern on "**" and match segments recursively.
	// e.g. "src/**/test/*.go" → parts=["src/", "/test/*.go"]
	parts := strings.SplitN(pattern, "**", 2)
	prefix := parts[0] // e.g. "src/"
	suffix := parts[1] // e.g. "/test/*.go"

	// Strip trailing/leading separators from prefix/suffix for matching
	prefixClean := strings.TrimSuffix(prefix, "/")
	suffixClean := strings.TrimPrefix(suffix, "/")

	// Check prefix matches (or is empty)
	if prefixClean != "" {
		if !strings.HasPrefix(path, prefixClean+"/") && path != prefixClean {
			return false
		}
	}

	// If no suffix after **, any path with the prefix matches
	if suffixClean == "" {
		return true
	}

	// Check suffix: try matching suffixClean against every possible tail of path
	// e.g. for path "src/pkg/test/foo.go" and suffix "test/*.go",
	// try "src/pkg/test/foo.go", "pkg/test/foo.go", "test/foo.go", "foo.go"
	remaining := path
	if prefixClean != "" {
		remaining = strings.TrimPrefix(path, prefixClean+"/")
	}
	pathParts := strings.Split(remaining, "/")
	for i := 0; i < len(pathParts); i++ {
		candidate := strings.Join(pathParts[i:], "/")
		if matched, err := filepath.Match(suffixClean, candidate); err == nil && matched {
			return true
		}
		// Also try recursive match if suffix itself contains **
		if strings.Contains(suffixClean, "**") && globMatch(suffixClean, candidate) {
			return true
		}
	}

	return false
}
