package stream

import "golang.org/x/net/context"

const key = "stream"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Stream associated with this context.
func FromContext(c context.Context) Stream {
	return c.Value(key).(Stream)
}

// ToContext adds the Stream to this context if it supports the
// Setter interface.
func ToContext(c Setter, s Stream) {
	c.Set(key, s)
}
