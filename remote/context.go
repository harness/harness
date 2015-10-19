package remote

import (
	"code.google.com/p/go.net/context"
)

const reqkey = "remote"

// NewContext returns a Context whose Value method returns
// the applications Remote instance.
func NewContext(parent context.Context, v Remote) context.Context {
	return &wrapper{parent, v}
}

type wrapper struct {
	context.Context
	v Remote
}

// Value returns the named key from the context.
func (c *wrapper) Value(key interface{}) interface{} {
	if key == reqkey {
		return c.v
	}
	return c.Context.Value(key)
}

// FromContext returns the Remote associated with this context.
func FromContext(c context.Context) Remote {
	return c.Value(reqkey).(Remote)
}
