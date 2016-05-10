package cache

import "sync"

// Memory provides an inmemory caching mechanism
type Memory struct {
	// Mutex is used for handling the concurrent
	// read/write requests for cache
	sync.Mutex

	// cache holds the cache data
	cache Cache
}

// NewMemory creates an inmemory cache system
// Which everytime will return the true value about a cache hit
func NewMemory() Cache {
	return &Memory{
		cache: NewMemoryNoTS(),
	}
}

// Get returns the value of a given key if it exists
func (r *Memory) Get(key string) (interface{}, error) {
	r.Lock()
	defer r.Unlock()

	return r.cache.Get(key)
}

// Set sets a value to the cache or overrides existing one with the given value
func (r *Memory) Set(key string, value interface{}) error {
	r.Lock()
	defer r.Unlock()

	return r.cache.Set(key, value)
}

// Delete deletes the given key-value pair from cache, this function doesnt
// return an error if item is not in the cache
func (r *Memory) Delete(key string) error {
	r.Lock()
	defer r.Unlock()

	return r.cache.Delete(key)
}
