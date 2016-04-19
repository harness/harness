package queue

import (
	"golang.org/x/net/context"
)

const key = "queue"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Queue associated with this context.
func FromContext(c context.Context) Queue {
	return c.Value(key).(Queue)
}

// ToContext adds the Queue to this context if it supports
// the Setter interface.
func ToContext(c Setter, q Queue) {
	c.Set(key, q)
}
