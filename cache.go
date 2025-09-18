package termenv

import (
	"container/list"
	"sync"
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

// Hasher is an interface that requires a Hash method. The Hash method is
// expected to return a string representation of the hash of the object.
type Hasher interface {
	Hash() string
}

// SequenceEntry is a struct that holds a key-value pair. It is used as an element
// in the evictionList of the Cache.
type SequenceEntry struct {
	key   [32]byte
	value string
}

// RGBCache is a struct that represents a cache with a set capacity. It
// uses an LRU (Least Recently Used) eviction policy. It is safe for
// concurrent use.
type RGBCache struct {
	capacity     int
	mutex        sync.Mutex
	cache        map[[32]byte]*list.Element // The cache holding the results
	evictionList *list.List                 // A list to keep track of the order for LRU
}

// NewRGBCache is a function that creates a new Cache with a given
// capacity. It returns a pointer to the created Cache.
func NewRGBCache(capacity int) *RGBCache {
	return &RGBCache{
		capacity:     capacity,
		cache:        make(map[[32]byte]*list.Element),
		evictionList: list.New(),
	}
}

// Capacity is a method that returns the capacity of the Cache.
func (m *RGBCache) Capacity() int {
	return m.capacity
}

// Size is a method that returns the current size of the Cache. It is
// the number of items currently stored in the cache.
func (m *RGBCache) Size() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.evictionList.Len()
}

// Get is a method that returns the value associated with the given
// hashable item in the Cache. If there is no corresponding value, the
// method returns nil.
func (m *RGBCache) Get(h RGBColor) (string, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	hashedKey := h.Hash()
	if element, found := m.cache[hashedKey]; found {
		m.evictionList.MoveToFront(element)
		return element.Value.(*SequenceEntry).value, true
	}
	var result string
	return result, false
}

// Set is a method that sets the value for the given hashable item in the
// Cache. If the cache is at capacity, it evicts the least recently
// used item before adding the new item.
func (m *RGBCache) Set(h RGBColor, value string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	hashedKey := h.Hash()
	if element, found := m.cache[hashedKey]; found {
		m.evictionList.MoveToFront(element)
		element.Value.(*SequenceEntry).value = value
		return
	}

	// Check if the cache is at capacity
	if m.evictionList.Len() >= m.capacity {
		// Evict the least recently used item from the cache
		toEvict := m.evictionList.Back()
		if toEvict != nil {
			evictedEntry := m.evictionList.Remove(toEvict).(*SequenceEntry)
			delete(m.cache, evictedEntry.key)
		}
	}

	// Add the value to the cache and the evictionList
	newEntry := &SequenceEntry{
		key:   hashedKey,
		value: value,
	}
	element := m.evictionList.PushFront(newEntry)
	m.cache[hashedKey] = element
}
