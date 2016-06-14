package cache

//go:generate mockery -name Cache -output mock -case=underscore

import (
	"time"

	"github.com/koding/cache"
	"golang.org/x/net/context"
)

type Cache interface {
	Get(string) (interface{}, error)
	Set(string, interface{}) error
	Delete(string) error
}

func Get(c context.Context, key string) (interface{}, error) {
	return FromContext(c).Get(key)
}

func Set(c context.Context, key string, value interface{}) error {
	return FromContext(c).Set(key, value)
}

func Delete(c context.Context, key string) error {
	return FromContext(c).Delete(key)
}

// Default creates an in-memory cache with the default
// 30 minute expiration period.
func Default() Cache {
	return NewTTL(time.Minute * 30)
}

// NewTTL returns an in-memory cache with the specified
// ttl expiration period.
func NewTTL(t time.Duration) Cache {
	return cache.NewMemoryWithTTL(t)
}
