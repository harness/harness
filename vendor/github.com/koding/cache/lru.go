package cache

import "sync"

// LRU Discards the least recently used items first. This algorithm
// requires keeping track of what was used when.
type LRU struct {
	// Mutex is used for handling the concurrent
	// read/write requests for cache
	sync.Mutex

	// cache holds the all cache values
	cache Cache
}

// NewLRU creates a thread-safe LRU cache
func NewLRU(size int) Cache {
	return &LRU{
		cache: NewLRUNoTS(size),
	}
}

// Get returns the value of a given key if it exists, every get item will be
// moved to the head of the linked list for keeping track of least recent used
// item
func (l *LRU) Get(key string) (interface{}, error) {
	l.Lock()
	defer l.Unlock()

	return l.cache.Get(key)
}

// Set sets or overrides the given key with the given value, every set item will
// be moved or prepended to the head of the linked list for keeping track of
// least recent used item. When the cache is full, last item of the linked list
// will be evicted from the cache
func (l *LRU) Set(key string, val interface{}) error {
	l.Lock()
	defer l.Unlock()

	return l.cache.Set(key, val)
}

// Delete deletes the given key-value pair from cache, this function doesnt
// return an error if item is not in the cache
func (l *LRU) Delete(key string) error {
	l.Lock()
	defer l.Unlock()

	return l.cache.Delete(key)
}
