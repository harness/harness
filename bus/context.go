package bus

import "golang.org/x/net/context"

const key = "bus"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Bus associated with this context.
func FromContext(c context.Context) Bus {
	return c.Value(key).(Bus)
}

// ToContext adds the Bus to this context if it supports
// the Setter interface.
func ToContext(c Setter, b Bus) {
	c.Set(key, b)
}
