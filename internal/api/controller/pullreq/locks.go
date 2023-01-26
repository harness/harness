// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"strconv"

	"github.com/harness/gitness/lock"
)

func (c *Controller) newMutexForPR(repoUID string, pr int64, options ...lock.Option) (lock.Mutex, error) {
	key := repoUID + "/pulls"
	if pr != 0 {
		key += "/" + strconv.FormatInt(pr, 10)
	}
	return c.mtxManager.NewMutex(key, append(options, lock.WithNamespace("repo"))...)
}
