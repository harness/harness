package cache

import (
	"time"

	"github.com/hashicorp/golang-lru"
)

// single instance of a thread-safe lru cache
var cache *lru.Cache

func init() {
	var err error
	cache, err = lru.New(2048)
	if err != nil {
		panic(err)
	}
}

// item is a simple wrapper around a cacheable object
// that tracks the ttl for item expiration in the cache.
type item struct {
	value interface{}
	ttl   time.Time
}

// set adds the key value pair to the cache with the
// specified ttl expiration.
func set(key string, value interface{}, ttl int64) {
	ttlv := time.Now().Add(time.Duration(ttl) * time.Second)
	cache.Add(key, &item{value, ttlv})
}

// get gets the value from the cache for the given key.
// if the value does not exist, a nil value is returned.
// if the value exists, but is expired, the value is returned
// with a bool flag set to true.
func get(key string) (interface{}, bool) {
	v, ok := cache.Get(key)
	if !ok {
		return nil, false
	}
	vv := v.(*item)
	expired := vv.ttl.Before(time.Now())
	if expired {
		cache.Remove(key)
	}
	return vv.value, expired
}
