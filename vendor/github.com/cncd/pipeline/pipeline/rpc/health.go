package rpc

import (
	"context"
)

// Health defines a health-check connection.
type Health interface {
  // Check returns if server is healthy or not
	Check(c context.Context) (bool, error)
}
