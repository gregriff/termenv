package termenv

import (
	"sync"

	"github.com/lucasb-eyer/go-colorful"
)

// init creates the RGB cache singletons
func init() {
	GetANSICache()
	GetSRGBCache()
}

var (
	ansiCache *RGBCache[string]
	sRGBCache *RGBCache[colorful.Color]
	ansiCacheInit,
	sRGBCacheInit sync.Once
)

// GetANSICache returns the global RGBColor->ANSI sequence cache instance.
// For use by Style.Foreground, this cache maps RGBColor's to ANSI sequences
func GetANSICache() *RGBCache[string] {
	ansiCacheInit.Do(func() {
		ansiCache = NewRGBCache[string](20)
	})
	return ansiCache
}

// GetSRGBCache returns the global RGBColor->sRGB cache instance.
// For use by Style.Styled, this cache maps RGBColor's to colorful.Color structs (stores sRGB data)
func GetSRGBCache() *RGBCache[colorful.Color] {
	sRGBCacheInit.Do(func() {
		sRGBCache = NewRGBCache[colorful.Color](20)
	})
	return sRGBCache
}

// RGBCache caches computed data given an RGBColor. Not safe for concurrent use.
// I added this because my TUI application renders markdown text with glamour (which calls funcs in this package)
// many times per second over and over again. Since this is the main functionality of my TUI, I profiled this feature
// and there were 3 funcs in termenv that were using much of the CPU time. Once I realized that my TUI would only ever
// need a fixed number of terminal colors/styles (computed by this package every time glamour renders markdown), I figured
// I'd create a cache for these. These caches (and one other perf tweak) led to almost a 2x reduction in CPU time for
// the code-path I was targeting, and a 5x speedup in the direct callee of these termenv functions I modified.
type RGBCache[T any] struct {
	data map[RGBColor]entry[T]

	capacity,
	counter int64
}

type entry[T any] struct {
	value      T
	lastAccess int64
}

func NewRGBCache[T any](capacity int) *RGBCache[T] {
	return &RGBCache[T]{
		data:     make(map[RGBColor]entry[T], capacity),
		capacity: int64(capacity),
	}
}

// Get retrieves a value if key is present and increases the total access count by one
func (c *RGBCache[T]) Get(key RGBColor) (T, bool) {
	e, ok := c.data[key]
	if !ok {
		var zero T
		return zero, false
	}

	c.counter += 1
	e.lastAccess = c.counter
	return e.value, true
}

// Put places a key into the cache if its not already there. It also increments the entry's counter
func (c *RGBCache[T]) Put(key RGBColor, value T) {
	c.counter += 1
	accessNum := c.counter

	if e, ok := c.data[key]; ok {
		e.value = value
		e.lastAccess = accessNum
		return
	}

	newEntry := &entry[T]{
		value:      value,
		lastAccess: accessNum,
	}
	c.data[key] = *newEntry

	// Check if we need to evict
	if int64(len(c.data)) >= c.capacity {
		c.evictLRU()
	}
}

// evictLRU performs O(n) eviction - finds and removes the least recently used entry
func (c *RGBCache[T]) evictLRU() {
	var oldestKey RGBColor
	var oldestAccess int64 = c.counter + 1 // start with max

	for key, value := range c.data {
		lastAccess := value.lastAccess

		if lastAccess < oldestAccess {
			oldestAccess = lastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}
