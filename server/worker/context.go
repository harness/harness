package worker

import (
	"code.google.com/p/go.net/context"
)

const reqkey = "worker"

// NewContext returns a Context whose Value method returns the
// application's worker queue.
func NewContext(parent context.Context, worker Worker) context.Context {
	return &wrapper{parent, worker}
}

type wrapper struct {
	context.Context
	worker Worker
}

// Value returns the named key from the context.
func (c *wrapper) Value(key interface{}) interface{} {
	if key == reqkey {
		return c.worker
	}
	return c.Context.Value(key)
}

// FromContext returns the worker queue associated with this context.
func FromContext(c context.Context) Worker {
	return c.Value(reqkey).(Worker)
}
