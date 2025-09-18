package termenv

import (
	"sync"
	"sync/atomic"
	"time"
)

func init() {
	GetRGBCache()
}

var (
	globalRGBCache *RGBCache
	once           sync.Once
)

// GetRGBCache returns the global RGB cache instance.
// It initializes the cache with default capacity on first call.
func GetRGBCache() *RGBCache {
	once.Do(func() {
		globalRGBCache = NewRGBCache(10)
	})
	return globalRGBCache
}

type RGBCache struct {
	data     sync.Map
	capacity int
	size     int64 // atomic counter
}

type entry struct {
	value    string
	lastUsed int64 // nanosecond timestamp for better precision
}

func NewRGBCache(capacity int) *RGBCache {
	return &RGBCache{
		capacity: capacity,
	}
}

func (c *RGBCache) Get(key RGBColor) (string, bool) {
	val, ok := c.data.Load(key)
	if !ok {
		return "", false
	}

	e := val.(*entry)
	// Update access time
	atomic.StoreInt64(&e.lastUsed, time.Now().UnixNano())

	return e.value, true
}

func (c *RGBCache) Put(key RGBColor, value string) {
	now := time.Now().UnixNano()

	// Check if key already exists
	if val, ok := c.data.Load(key); ok {
		// Update existing entry
		e := val.(*entry)
		e.value = value
		atomic.StoreInt64(&e.lastUsed, now)
		return
	}

	// New entry
	newEntry := &entry{
		value:    value,
		lastUsed: now,
	}

	c.data.Store(key, newEntry)
	newSize := atomic.AddInt64(&c.size, 1)

	// Check if we need to evict
	if int(newSize) > c.capacity {
		c.evictLRU()
	}
}

func (c *RGBCache) Delete(key RGBColor) bool {
	_, existed := c.data.LoadAndDelete(key)
	if existed {
		atomic.AddInt64(&c.size, -1)
	}
	return existed
}

func (c *RGBCache) Len() int {
	return int(atomic.LoadInt64(&c.size))
}

func (c *RGBCache) Clear() {
	c.data.Range(func(key, value interface{}) bool {
		c.data.Delete(key)
		return true
	})
	atomic.StoreInt64(&c.size, 0)
}

// O(n) eviction - find and remove the least recently used entry
func (c *RGBCache) evictLRU() {
	var oldestKey interface{}
	var oldestTime int64 = time.Now().UnixNano()

	c.data.Range(func(key, value interface{}) bool {
		e := value.(*entry)
		lastUsed := atomic.LoadInt64(&e.lastUsed)

		if lastUsed < oldestTime {
			oldestTime = lastUsed
			oldestKey = key
		}
		return true
	})

	if oldestKey != nil {
		c.data.Delete(oldestKey)
		atomic.AddInt64(&c.size, -1)
	}
}
