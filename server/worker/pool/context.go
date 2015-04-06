package pool

import (
	"code.google.com/p/go.net/context"
)

const reqkey = "pool"

// NewContext returns a Context whose Value method returns the
// worker pool.
func NewContext(parent context.Context, pool *Pool) context.Context {
	return &wrapper{parent, pool}
}

type wrapper struct {
	context.Context
	pool *Pool
}

// Value returns the named key from the context.
func (c *wrapper) Value(key interface{}) interface{} {
	if key == reqkey {
		return c.pool
	}
	return c.Context.Value(key)
}

// FromContext returns the pool assigned to the context.
func FromContext(c context.Context) *Pool {
	return c.Value(reqkey).(*Pool)
}
