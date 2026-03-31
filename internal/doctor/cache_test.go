package doctor

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheKey_Deterministic(t *testing.T) {
	key1 := CacheKey([]byte("content"), "prompt", "model", "v1")
	key2 := CacheKey([]byte("content"), "prompt", "model", "v1")
	assert.Equal(t, key1, key2)
	assert.Len(t, key1, 16)
}

func TestCacheKey_DifferentContent(t *testing.T) {
	key1 := CacheKey([]byte("content-a"), "prompt", "model", "v1")
	key2 := CacheKey([]byte("content-b"), "prompt", "model", "v1")
	assert.NotEqual(t, key1, key2)
}

func TestCacheKey_DifferentModel(t *testing.T) {
	key1 := CacheKey([]byte("content"), "prompt", "model-a", "v1")
	key2 := CacheKey([]byte("content"), "prompt", "model-b", "v1")
	assert.NotEqual(t, key1, key2)
}

func TestCacheKey_DifferentRegistry(t *testing.T) {
	key1 := CacheKey([]byte("content"), "prompt", "model", "v1")
	key2 := CacheKey([]byte("content"), "prompt", "model", "v2")
	assert.NotEqual(t, key1, key2)
}

func TestCache_PutAndGet(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(filepath.Join(dir, "cache"))

	findings := []llmutil.Finding{
		{Category: "guardrails", Issue: "No timeout", Confidence: "high"},
	}
	key := "test-key-12345678"
	entry := CacheEntry{
		ContentHash:     "abc123",
		Model:           "llama4",
		RegistryVersion: "2026.03.2",
		Timestamp:       time.Now(),
		Findings:        findings,
	}

	err := c.Put(key, entry)
	require.NoError(t, err)

	got := c.Get(key)
	require.NotNil(t, got)
	assert.Len(t, got, 1)
	assert.Equal(t, "guardrails", got[0].Category)
	assert.Equal(t, "No timeout", got[0].Issue)
}

func TestCache_Miss_NoFile(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(filepath.Join(dir, "cache"))
	got := c.Get("nonexistent-key")
	assert.Nil(t, got)
}

func TestCache_Miss_Expired(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(filepath.Join(dir, "cache"))

	key := "expired-key-0000"
	entry := CacheEntry{
		ContentHash: "abc",
		Timestamp:   time.Now().Add(-8 * 24 * time.Hour), // 8 days ago
		Findings:    []llmutil.Finding{{Category: "test", Issue: "old"}},
	}

	err := c.Put(key, entry)
	require.NoError(t, err)

	got := c.Get(key)
	assert.Nil(t, got, "expired entries should return nil")
}

func TestCache_Miss_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	require.NoError(t, os.MkdirAll(cacheDir, 0755))

	// Write invalid JSON
	path := filepath.Join(cacheDir, "bad-json-key.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid"), 0644))

	c := NewCache(cacheDir)
	got := c.Get("bad-json-key")
	assert.Nil(t, got)
}

func TestCache_Purge(t *testing.T) {
	dir := t.TempDir()
	c := NewCache(filepath.Join(dir, "cache"))

	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte("my-file-content")))[:16]

	// Put two entries with same content hash, one with different
	entry1 := CacheEntry{ContentHash: contentHash, Timestamp: time.Now(), Findings: []llmutil.Finding{{Issue: "a"}}}
	entry2 := CacheEntry{ContentHash: contentHash, Timestamp: time.Now(), Findings: []llmutil.Finding{{Issue: "b"}}}
	entry3 := CacheEntry{ContentHash: "different-hash", Timestamp: time.Now(), Findings: []llmutil.Finding{{Issue: "c"}}}

	require.NoError(t, c.Put("key-1", entry1))
	require.NoError(t, c.Put("key-2", entry2))
	require.NoError(t, c.Put("key-3", entry3))

	// Purge entries with matching content hash
	c.Purge(contentHash)

	assert.Nil(t, c.Get("key-1"), "purged entry should be nil")
	assert.Nil(t, c.Get("key-2"), "purged entry should be nil")
	assert.NotNil(t, c.Get("key-3"), "non-matching entry should survive purge")
}

func TestCache_CreatesDirOnPut(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "deep", "nested", "cache")
	c := NewCache(cacheDir)

	entry := CacheEntry{Timestamp: time.Now(), Findings: []llmutil.Finding{}}
	err := c.Put("test-key", entry)
	require.NoError(t, err)

	_, err = os.Stat(cacheDir)
	assert.NoError(t, err, "cache dir should be created")
}
