package cache

import (
	"golang.org/x/net/context"
)

const key = "cache"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Cache associated with this context.
func FromContext(c context.Context) Cache {
	return c.Value(key).(Cache)
}

// ToContext adds the Cache to this context if it supports
// the Setter interface.
func ToContext(c Setter, cache Cache) {
	c.Set(key, cache)
}
