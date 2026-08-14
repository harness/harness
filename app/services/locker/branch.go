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
	"time"
)

// LockBranch is used by merge service and API to lock changing of a PR's target branch.
func (l Locker) LockBranch(
	ctx context.Context,
	repoID int64,
	branch string,
	expiry time.Duration,
) (func(), error) {
	key := fmt.Sprintf("%d/branch/%s", repoID, branch)

	unlockFn, err := l.lock(ctx, namespaceRepo, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to lock mutex for branch %q in repo %d: %w", branch, repoID, err)
	}

	return unlockFn, nil
}
