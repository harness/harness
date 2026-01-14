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

	"github.com/rs/zerolog/log"
)

func (l Locker) LockDefaultBranch(
	ctx context.Context,
	repoID int64,
	branchName string,
	expiry time.Duration,
) (func(), error) {
	key := strconv.FormatInt(repoID, 10) + "/defaultBranch"

	log.Ctx(ctx).Info().Msg("attempting to lock to update the repo default branch")

	unlockFn, err := l.lock(ctx, namespaceRepo, key, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to lock repo to update default branch to %s: %w", branchName, err)
	}

	return unlockFn, nil
}
