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

package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/contextutil"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func (c *Controller) lockDefaultBranch(
	ctx context.Context,
	repoUID string,
	branchName string,
	expiry time.Duration,
) (func(), error) {
	key := repoUID + "/defaultBranch"

	// annotate logs for easier debugging of lock related issues
	// TODO: refactor once common logging annotations are added
	ctx = logging.NewContext(ctx, func(zc zerolog.Context) zerolog.Context {
		return zc.
			Str("default_branch_lock", key).
			Str("repo_uid", repoUID)
	})

	mutext, err := c.mtxManager.NewMutex(
		key,
		lock.WithNamespace("repo"),
		lock.WithExpiry(expiry),
		lock.WithTimeoutFactor(4/expiry.Seconds()), // 4s
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new mutex for repo %q with default branch %s: %w", repoUID, branchName, err)
	}

	err = mutext.Lock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to lock the mutex for repo %q with default branch %s: %w", repoUID, branchName, err)
	}

	log.Ctx(ctx).Info().Msgf("successfully locked the repo default branch (expiry: %s)", expiry)

	unlockFn := func() {
		// always unlock independent of whether source context got canceled or not
		ctx, cancel := context.WithTimeout(
			contextutil.WithNewValues(context.Background(), ctx),
			30*time.Second,
		)
		defer cancel()

		err := mutext.Unlock(ctx)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to unlock repo default branch")
		} else {
			log.Ctx(ctx).Info().Msg("successfully unlocked repo default branch")
		}
	}
	return unlockFn, nil
}
