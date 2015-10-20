package cache

import (
	"time"

	"github.com/koding/cache"
	"golang.org/x/net/context"
)

type Cache interface {
	Get(string) (interface{}, error)
	Set(string, interface{}) error
}

func Get(c context.Context, key string) (interface{}, error) {
	return FromContext(c).Get(key)
}

func Set(c context.Context, key string, value interface{}) error {
	return FromContext(c).Set(key, value)
}

// Default creates an in-memory cache with the default
// 24 hour expiration period.
func Default() Cache {
	return cache.NewMemoryWithTTL(time.Hour * 24)
}

// NewTTL returns an in-memory cache with the specified
// ttl expiration period.
func NewTTL(t time.Duration) Cache {
	return cache.NewMemoryWithTTL(t)
}

// NewTTL returns an in-memory cache with the specified
// ttl expiration period.
func NewLRU(size int) Cache {
	return cache.NewLRU(size)
}
