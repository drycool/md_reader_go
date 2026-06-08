// Package cache provides caching implementations for md-reader.
// Supports in-memory LRU cache and file-based persistent cache.
package cache

import (
	"container/list"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/drycool/md_reader_go/internal/exceptions"
	"github.com/drycool/md_reader_go/internal/logger"
)

// CacheEntry holds a cached value with metadata.
type CacheEntry struct {
	Value      interface{}
	CreatedAt  time.Time
	LastAccess time.Time
	AccessCount int
	TTL        time.Duration // 0 means no TTL
}

// IsExpired checks if the entry has exceeded its TTL.
func (e *CacheEntry) IsExpired() bool {
	if e.TTL == 0 {
		return false
	}
	return time.Since(e.CreatedAt) > e.TTL
}

// Touch updates the access time and count.
func (e *CacheEntry) Touch() {
	e.LastAccess = time.Now()
	e.AccessCount++
}

// CacheStats holds cache performance statistics.
type CacheStats struct {
	Size        int
	MaxSize     int
	HitRate     float64
	Hits        int64
	Misses      int64
	Evictions   int64
	Expirations int64
}

// Cache defines the interface for cache implementations.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Invalidate(key string)
	Clear()
	Stats() CacheStats
}

// --- Memory Cache (LRU) ---

// MemoryCache is an in-memory LRU cache with TTL support.
type MemoryCache struct {
	mu       sync.RWMutex
	maxSize  int
	defaultTTL time.Duration
	items    map[string]*list.Element
	order    *list.List
	stats    CacheStats
}

type cacheItem struct {
	key   string
	entry CacheEntry
}

// NewMemoryCache creates a new MemoryCache.
// maxSize: maximum number of entries (0 = unlimited)
// defaultTTL: default TTL for entries (0 = no expiry)
func NewMemoryCache(maxSize int, defaultTTL time.Duration) *MemoryCache {
	return &MemoryCache{
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		items:      make(map[string]*list.Element),
		order:      list.New(),
	}
}

// Get retrieves a value from cache.
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		c.stats.Misses++
		return nil, false
	}

	item := elem.Value.(*cacheItem)

	// Check expiry
	if item.entry.IsExpired() {
		c.order.Remove(elem)
		delete(c.items, key)
		c.stats.Expirations++
		c.stats.Misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	item.entry.Touch()
	c.stats.Hits++

	return item.entry.Value, true
}

// Set stores a value in cache.
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key exists, update it
	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*cacheItem)
		item.entry.Value = value
		item.entry.CreatedAt = time.Now()
		item.entry.LastAccess = time.Now()
		item.entry.AccessCount = 0
		if ttl > 0 {
			item.entry.TTL = ttl
		} else if c.defaultTTL > 0 {
			item.entry.TTL = c.defaultTTL
		} else {
			item.entry.TTL = 0
		}
		c.order.MoveToFront(elem)
		return
	}

	// Evict if needed
	if c.maxSize > 0 && len(c.items) >= c.maxSize {
		c.evictLRU()
	}

	// Create new entry
	entry := CacheEntry{
		Value:     value,
		CreatedAt: time.Now(),
		LastAccess: time.Now(),
	}
	if ttl > 0 {
		entry.TTL = ttl
	} else {
		entry.TTL = c.defaultTTL
	}

	elem := c.order.PushFront(&cacheItem{key: key, entry: entry})
	c.items[key] = elem
}

// Invalidate removes a key from cache.
func (c *MemoryCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.order.Remove(elem)
		delete(c.items, key)
	}
}

// Clear empties the entire cache.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
	c.stats = CacheStats{MaxSize: c.maxSize}
}

// Stats returns cache performance statistics.
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.stats.Hits) / float64(total)
	}

	stats := c.stats
	stats.Size = len(c.items)
	stats.MaxSize = c.maxSize
	stats.HitRate = hitRate
	return stats
}

func (c *MemoryCache) evictLRU() {
	elem := c.order.Back()
	if elem == nil {
		return
	}
	item := elem.Value.(*cacheItem)
	delete(c.items, item.key)
	c.order.Remove(elem)
	c.stats.Evictions++
}

// --- File-Based Cache ---

// FileCache is a file-based persistent cache.
// Entries are serialized with gob encoding.
type FileCache struct {
	mu        sync.RWMutex
	cacheDir  string
	maxSize   int
	defaultTTL time.Duration
	stats     CacheStats
}

// NewFileCache creates a new FileCache.
func NewFileCache(cacheDir string, maxSizeMB int, defaultTTL time.Duration) *FileCache {
	absDir, err := filepath.Abs(cacheDir)
	if err != nil {
		absDir = cacheDir
	}
	os.MkdirAll(absDir, 0755)

	// Register types for gob encoding
	gob.Register(CacheEntry{})

	return &FileCache{
		cacheDir:   absDir,
		maxSize:    maxSizeMB,
		defaultTTL:  defaultTTL,
	}
}

func (fc *FileCache) cachePath(key string) string {
	// Use a simple hash/encoding for the filename
	return filepath.Join(fc.cacheDir, sanitizeKey(key))
}

// Get retrieves a value from file cache.
func (fc *FileCache) Get(key string) (interface{}, bool) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	cachePath := fc.cachePath(key)

	f, err := os.Open(cachePath)
	if err != nil {
		fc.stats.Misses++
		return nil, false
	}
	defer f.Close()

	var entry CacheEntry
	decoder := gob.NewDecoder(f)
	if err := decoder.Decode(&entry); err != nil {
		fc.stats.Misses++
		return nil, false
	}

	if entry.IsExpired() {
		os.Remove(cachePath)
		fc.stats.Expirations++
		fc.stats.Misses++
		return nil, false
	}

	entry.Touch()
	fc.stats.Hits++

	// Update access time (best-effort)
	if err := fc.writeEntry(cachePath, &entry); err != nil {
		// Silent failure — the value is still valid for this call
	}

	return entry.Value, true
}

// Set stores a value in file cache.
func (fc *FileCache) Set(key string, value interface{}, ttl time.Duration) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	cachePath := fc.cachePath(key)

	entry := CacheEntry{
		Value:      value,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
	}
	if ttl > 0 {
		entry.TTL = ttl
	} else {
		entry.TTL = fc.defaultTTL
	}

	if err := fc.writeEntry(cachePath, &entry); err != nil {
		logger.GetLogger("cache").Warn("Failed to write cache entry",
			"key", key,
			"error", err,
		)
	}
}

// Invalidate removes a key from file cache.
func (fc *FileCache) Invalidate(key string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	os.Remove(fc.cachePath(key))
}

// Clear removes all cache files.
func (fc *FileCache) Clear() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	entries, err := os.ReadDir(fc.cacheDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		os.Remove(filepath.Join(fc.cacheDir, entry.Name()))
	}
	fc.stats = CacheStats{}
}

// Stats returns file cache statistics.
func (fc *FileCache) Stats() CacheStats {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	total := fc.stats.Hits + fc.stats.Misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(fc.stats.Hits) / float64(total)
	}

	stats := fc.stats
	stats.HitRate = hitRate
	return stats
}

func (fc *FileCache) writeEntry(path string, entry *CacheEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating cache file: %w", err)
	}
	defer f.Close()

	encoder := gob.NewEncoder(f)
	return encoder.Encode(entry)
}

// sanitizeKey converts a key to a safe filename.
func sanitizeKey(key string) string {
	safe := make([]byte, 0, len(key))
	for _, c := range []byte(key) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			safe = append(safe, c)
		} else if c == '/' || c == '\\' {
			safe = append(safe, '_')
		}
		// Skip other characters
	}
	if len(safe) == 0 {
		return "_"
	}
	return string(safe)
}

// --- Tiered Cache ---

// TieredCache combines memory and file caching.
type TieredCache struct {
	memoryCache *MemoryCache
	fileCache   *FileCache
}

// NewTieredCache creates a tiered cache (memory first, then file).
func NewTieredCache(memoryMaxSize int, fileDir string, fileMaxSizeMB int, defaultTTL time.Duration) *TieredCache {
	return &TieredCache{
		memoryCache: NewMemoryCache(memoryMaxSize, defaultTTL),
		fileCache:   NewFileCache(fileDir, fileMaxSizeMB, defaultTTL),
	}
}

// Get checks memory first, then file cache.
func (tc *TieredCache) Get(key string) (interface{}, bool) {
	// Check memory first
	val, ok := tc.memoryCache.Get(key)
	if ok {
		return val, true
	}

	// Check file cache
	val, ok = tc.fileCache.Get(key)
	if ok {
		// Promote to memory
		tc.memoryCache.Set(key, val, 0)
		return val, true
	}

	return nil, false
}

// Set stores in both caches.
func (tc *TieredCache) Set(key string, value interface{}, ttl time.Duration) {
	tc.memoryCache.Set(key, value, ttl)
	tc.fileCache.Set(key, value, ttl)
}

// Invalidate removes from both caches.
func (tc *TieredCache) Invalidate(key string) {
	tc.memoryCache.Invalidate(key)
	tc.fileCache.Invalidate(key)
}

// Clear empties both caches.
func (tc *TieredCache) Clear() {
	tc.memoryCache.Clear()
	tc.fileCache.Clear()
}

// Stats returns combined stats (memory-focused).
func (tc *TieredCache) Stats() CacheStats {
	return tc.memoryCache.Stats()
}

// --- Factory ---

// NewCache creates a cache of the specified type.
func NewCache(cacheType string, opts ...int) (Cache, error) {
	switch cacheType {
	case "memory":
		maxSize := 1000
		if len(opts) > 0 {
			maxSize = opts[0]
		}
		return NewMemoryCache(maxSize, 5*time.Minute), nil
	case "file":
		fcache := NewFileCache(".cache", 100, 30*time.Minute)
		return fcache, nil
	case "tiered":
		return NewTieredCache(1000, ".cache", 100, 5*time.Minute), nil
	default:
		return nil, exceptions.NewConfigurationError("cache_type", "Unknown cache type: "+cacheType)
	}
}
