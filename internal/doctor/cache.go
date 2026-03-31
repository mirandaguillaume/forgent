// Package doctor provides the top-level doctor analysis orchestration.
package doctor

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
)

// DefaultCacheTTL is the time-to-live for cache entries.
const DefaultCacheTTL = 7 * 24 * time.Hour

// CacheEntry stores a cached LLM analysis result.
type CacheEntry struct {
	ContentHash     string           `json:"content_hash"`
	PromptHash      string           `json:"prompt_hash"`
	Model           string           `json:"model"`
	RegistryVersion string           `json:"registry_version"`
	Timestamp       time.Time        `json:"timestamp"`
	Findings        []llmutil.Finding `json:"findings"`
}

// Cache manages LLM response caching for doctor analysis.
type Cache struct {
	dir string
	ttl time.Duration
}

// NewCache creates a cache backed by the given directory.
// The directory is created on first write if it does not exist.
func NewCache(dir string) *Cache {
	return &Cache{dir: dir, ttl: DefaultCacheTTL}
}

// CacheKey computes a deterministic key from file content and analysis configuration.
// CacheKey computes a deterministic key from file content and analysis configuration.
// Fields are separated by null bytes to prevent concatenation collisions.
func CacheKey(content []byte, promptHash, model, registryVersion string) string {
	h := sha256.New()
	h.Write(content)
	h.Write([]byte{0})
	h.Write([]byte(promptHash))
	h.Write([]byte{0})
	h.Write([]byte(model))
	h.Write([]byte{0})
	h.Write([]byte(registryVersion))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// Get returns cached findings if valid, or nil on cache miss.
func (c *Cache) Get(key string) []llmutil.Finding {
	path := c.path(key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}

	if time.Since(entry.Timestamp) > c.ttl {
		return nil
	}

	return entry.Findings
}

// Put stores findings in the cache.
func (c *Cache) Put(key string, entry CacheEntry) error {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache entry: %w", err)
	}

	return os.WriteFile(c.path(key), data, 0600)
}

// Purge removes all cache entries whose content_hash matches the given hash.
// This is called after --fix modifies a file.
func (c *Cache) Purge(contentHash string) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(c.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		if entry.ContentHash == contentHash {
			os.Remove(path)
		}
	}
}

func (c *Cache) path(key string) string {
	return filepath.Join(c.dir, key+".json")
}
