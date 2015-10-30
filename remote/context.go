package remote

import (
	"golang.org/x/net/context"
)

const key = "remote"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Remote associated with this context.
func FromContext(c context.Context) Remote {
	return c.Value(key).(Remote)
}

// ToContext adds the Remote to this context if it supports
// the Setter interface.
func ToContext(c Setter, r Remote) {
	c.Set(key, r)
}
