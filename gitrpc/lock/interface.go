// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import "context"

// Locker acquires new lock based on key.
type Locker interface {
	AcquireLock(ctx context.Context, key string) (*Lock, error)
}
