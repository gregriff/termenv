package termenv

import (
	"sync"
	"sync/atomic"
)

// init creates the RGB cache singletons
func init() {
	GetANSICache()
	GetSRGBCache()
}

var (
	ansiCache,
	sRGBCache *RGBCache
	ansiCacheInit,
	sRGBCacheInit sync.Once
)

// GetANSICache returns the global RGBColor->ANSI sequence cache instance.
// For use by Style.Foreground, this cache maps RGBColor's to ANSI sequences
func GetANSICache() *RGBCache {
	ansiCacheInit.Do(func() {
		ansiCache = NewRGBCache(20)
	})
	return ansiCache
}

// GetSRGBCache returns the global RGBColor->sRGB cache instance.
// For use by Style.Styled, this cache maps RGBColor's to colorful.Color structs (stores sRGB data)
func GetSRGBCache() *RGBCache {
	sRGBCacheInit.Do(func() {
		sRGBCache = NewRGBCache(20)
	})
	return sRGBCache
}

// RGBCache caches computed data given an RGBColor.
// I added this because my TUI application renders markdown text with glamour (which calls funcs in this package)
// many times per second over and over again. Since this is the main functionality of my TUI, I profiled this feature
// and there were 3 funcs in termenv that were using much of the CPU time. Once I realized that my TUI would only ever
// need a fixed number of terminal colors/styles (computed by this package every time glamour renders markdown), I figured
// I'd create a cache for these. These caches (and one other perf tweak) led to almost a 2x reduction in CPU time for
// the code-path I was targeting, and a 5x speedup in the direct callee of these termenv functions I modified.
type RGBCache struct {
	data sync.Map

	capacity,
	size,
	counter int64 // atomic counters
}

type entry struct {
	value      interface{} // go 1.18 generics would be nice to have here!
	lastAccess int64
}

func NewRGBCache(capacity int) *RGBCache {
	return &RGBCache{
		capacity: int64(capacity),
	}
}

// Get retrieves a value if key is present and increases the total access count by one
func (c *RGBCache) Get(key RGBColor) (interface{}, bool) {
	val, ok := c.data.Load(key)
	if !ok {
		return "", false
	}

	e := val.(*entry)
	atomic.StoreInt64(&e.lastAccess, atomic.AddInt64(&c.counter, 1))

	return e.value, true
}

// Put places a key into the cache if its not already there. It also increments the entry's counter
func (c *RGBCache) Put(key RGBColor, value interface{}) {
	accessNum := atomic.AddInt64(&c.counter, 1)

	if val, ok := c.data.Load(key); ok {
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

// Delete removes an item from the cache
// func (c *RGBCache) Delete(key RGBColor) bool {
// 	_, existed := c.data.LoadAndDelete(key)
// 	if existed {
// 		atomic.AddInt64(&c.size, -1)
// 	}
// 	return existed
// }

// Len returns the number of items in the cache atomically
// func (c *RGBCache) Len() int {
// 	return int(atomic.LoadInt64(&c.size))
// }

// Clear empties the cache. Untested
// func (c *RGBCache) Clear() {
// 	c.data.Range(func(key, value interface{}) bool {
// 		c.data.Delete(key)
// 		return true
// 	})
// 	atomic.StoreInt64(&c.size, 0)
// }

// evictLRU performs O(n) eviction - finds and removes the least recently used entry
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
