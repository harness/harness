package lock

import "context"

// Locker acquires new lock based on key.
type Locker interface {
	AcquireLock(ctx context.Context, key string) (*Lock, error)
}
