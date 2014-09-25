package capability

import (
	"code.google.com/p/go.net/context"
)

const reqkey = "capability"

// NewContext returns a Context whose Value method returns the
// application's Blobstore data.
func NewContext(parent context.Context, caps Capability) context.Context {
	return &wrapper{parent, caps}
}

type wrapper struct {
	context.Context
	caps Capability
}

// Value returns the named key from the context.
func (c *wrapper) Value(key interface{}) interface{} {
	if key == reqkey {
		return c.caps
	}
	return c.Context.Value(key)
}

// FromContext returns the capability map for the
// current context.
func FromContext(c context.Context) Capability {
	return c.Value(reqkey).(Capability)
}
