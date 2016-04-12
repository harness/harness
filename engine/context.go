package engine

import (
	"golang.org/x/net/context"
)

const key = "engine"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Engine associated with this context.
func FromContext(c context.Context) Engine {
	return c.Value(key).(Engine)
}

// ToContext adds the Engine to this context if it supports
// the Setter interface.
func ToContext(c Setter, engine Engine) {
	c.Set(key, engine)
}
