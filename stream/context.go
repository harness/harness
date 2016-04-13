package stream

import "golang.org/x/net/context"

const key = "stream"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Mux associated with this context.
func FromContext(c context.Context) Mux {
	return c.Value(key).(Mux)
}

// ToContext adds the Mux to this context if it supports
// the Setter interface.
func ToContext(c Setter, m Mux) {
	c.Set(key, m)
}
