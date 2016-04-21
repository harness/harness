package docker

import (
	"github.com/drone/drone/engine/runner"
	"golang.org/x/net/context"
)

const key = "docker"

// Setter defines a context that enables setting values.
type Setter interface {
	Set(string, interface{})
}

// FromContext returns the Engine associated with this context.
func FromContext(c context.Context) runner.Engine {
	return c.Value(key).(runner.Engine)
}

// ToContext adds the Engine to this context if it supports the
// Setter interface.
func ToContext(c Setter, d runner.Engine) {
	c.Set(key, d)
}
