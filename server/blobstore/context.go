package blobstore

import (
	"code.google.com/p/go.net/context"
)

const reqkey = "blobstore"

// NewContext returns a Context whose Value method returns the
// application's Blobstore data.
func NewContext(parent context.Context, store Blobstore) context.Context {
	return &wrapper{parent, store}
}

type wrapper struct {
	context.Context
	store Blobstore
}

// Value returns the named key from the context.
func (c *wrapper) Value(key interface{}) interface{} {
	if key == reqkey {
		return c.store
	}
	return c.Context.Value(key)
}

// FromContext returns the Blobstore associated with this context.
func FromContext(c context.Context) Blobstore {
	return c.Value(reqkey).(Blobstore)
}
