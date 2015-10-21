package cache

import "testing"

func TestMemoryGetSet(t *testing.T) {
	cache := NewMemory()
	testCacheGetSet(t, cache)
}

func TestMemoryDelete(t *testing.T) {
	cache := NewMemory()
	testCacheDelete(t, cache)
}

func TestMemoryNilValue(t *testing.T) {
	cache := NewMemory()
	testCacheNilValue(t, cache)
}
