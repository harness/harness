// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package locker

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

func (l Locker) LockPR(
	ctx context.Context,
	repoID int64,
	prNum int64,
	expiry time.Duration,
) (func(), error) {
	key := fmt.Sprintf("%d/pulls", repoID)
	if prNum != 0 {
		key += "/" + strconv.FormatInt(prNum, 10)
	}

	unlockFn, err := l.lock(ctx, namespaceRepo, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to lock mutex for pr %d in repo %d: %w", prNum, repoID, err)
	}

	return unlockFn, nil
}
