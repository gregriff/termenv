package termenv

import (
	"sync"
	"sync/atomic"
)

func init() {
	GetRGBSequenceCache()
	GetRGBHexCache()
}

var (
	globalRGBSequenceCache,
	globalRGBHexCache *RGBCache
	sequenceOnce,
	hexOnce sync.Once
)

// GetRGBSequenceCache returns the global RGB sequence cache instance.
// for use by Style.Foreground
func GetRGBSequenceCache() *RGBCache {
	sequenceOnce.Do(func() {
		globalRGBSequenceCache = NewRGBCache(20)
	})
	return globalRGBSequenceCache
}

func GetRGBHexCache() *RGBCache {
	hexOnce.Do(func() {
		globalRGBHexCache = NewRGBCache(20)
	})
	return globalRGBHexCache
}

type RGBCache struct {
	data sync.Map
	capacity,
	size,
	counter int64 // atomic counters
}

type entry struct {
	value      interface{}
	lastAccess int64
}

func NewRGBCache(capacity int) *RGBCache {
	return &RGBCache{
		capacity: int64(capacity),
	}
}

func (c *RGBCache) Get(key string) (interface{}, bool) {
	val, ok := c.data.Load(key)
	if !ok {
		return "", false
	}

	e := val.(*entry)
	atomic.StoreInt64(&e.lastAccess, atomic.AddInt64(&c.counter, 1))

	return e.value, true
}

func (c *RGBCache) Put(key string, value interface{}) {
	accessNum := atomic.AddInt64(&c.counter, 1)

	// Check if key already exists
	if val, ok := c.data.Load(key); ok {
		// Update existing entry
		e := val.(*entry)
		e.value = value
		atomic.StoreInt64(&e.lastAccess, accessNum)
		return
	}

	// New entry
	newEntry := &entry{
		value:      value,
		lastAccess: accessNum,
	}

	c.data.Store(key, newEntry)
	newSize := atomic.AddInt64(&c.size, 1)

	// Check if we need to evict
	if newSize > c.capacity {
		c.evictLRU()
	}
}

func (c *RGBCache) Delete(key string) bool {
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
	var oldestAccess int64 = atomic.LoadInt64(&c.counter) + 1 // start with max

	c.data.Range(func(key, value interface{}) bool {
		e := value.(*entry)
		lastAccess := atomic.LoadInt64(&e.lastAccess)

		if lastAccess < oldestAccess {
			oldestAccess = lastAccess
			oldestKey = key
		}
		return true
	})

	if oldestKey != nil {
		c.data.Delete(oldestKey)
		atomic.AddInt64(&c.size, -1)
	}
}
