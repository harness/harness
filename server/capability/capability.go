package capability

import (
	"code.google.com/p/go.net/context"
)

type Capability map[string]bool

// Get the capability value from the map.
func (c Capability) Get(key string) bool {
	return c.Get(key)
}

// Sets the capability value in the map.
func (c Capability) Set(key string, value bool) {
	c[key] = value
}

// Enabled returns true if the capability is
// enabled in the system.
func Enabled(c context.Context, key string) bool {
	return FromContext(c).Get(key)
}
