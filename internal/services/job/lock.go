// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package job

import (
	"context"

	"github.com/harness/gitness/lock"
)

func globalLock(ctx context.Context, manager lock.MutexManager) (lock.Mutex, error) {
	const lockKey = "jobs"
	mx, err := manager.NewMutex(lockKey)
	if err != nil {
		return nil, err
	}

	err = mx.Lock(ctx)

	return mx, err
}
