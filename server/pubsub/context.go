package pubsub

import (
	"code.google.com/p/go.net/context"
)

const reqkey = "pubsub"

// NewContext returns a Context whose Value method returns the
// PubSub module.
func NewContext(parent context.Context, pubsub *PubSub) context.Context {
	return &wrapper{parent, pubsub}
}

type wrapper struct {
	context.Context
	pubsub *PubSub
}

// Value returns the named key from the context.
func (c *wrapper) Value(key interface{}) interface{} {
	if key == reqkey {
		return c.pubsub
	}
	return c.Context.Value(key)
}

// FromContext returns the pool assigned to the context.
func FromContext(c context.Context) *PubSub {
	return c.Value(reqkey).(*PubSub)
}
