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

package pullreq

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/harness/gitness/contextutil"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func (c *Controller) lockPR(
	ctx context.Context,
	repoID int64,
	prNum int64,
	expiry time.Duration,
) (func(), error) {
	key := fmt.Sprintf("%d/pulls", repoID)
	if prNum != 0 {
		key += "/" + strconv.FormatInt(prNum, 10)
	}

	// annotate logs for easier debugging of lock related merge issues
	// TODO: refactor once common logging annotations are added
	ctx = logging.NewContext(ctx, func(c zerolog.Context) zerolog.Context {
		return c.
			Str("pullreq_lock", key).
			Int64("repo_id", repoID)
	})

	mutex, err := c.mtxManager.NewMutex(
		key,
		lock.WithNamespace("repo"),
		lock.WithExpiry(expiry),
		lock.WithTimeoutFactor(4/expiry.Seconds()), // 4s
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new mutex for pr %d in repo %q: %w", prNum, repoID, err)
	}
	err = mutex.Lock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to lock mutex for pr %d in repo %q: %w", prNum, repoID, err)
	}

	log.Ctx(ctx).Debug().Msgf("successfully locked PR (expiry: %s)", expiry)

	unlockFn := func() {
		// always unlock independent of whether source context got canceled or not
		ctx, cancel := context.WithTimeout(
			contextutil.WithNewValues(context.Background(), ctx),
			30*time.Second,
		)
		defer cancel()

		err := mutex.Unlock(ctx)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to unlock PR")
		} else {
			log.Ctx(ctx).Debug().Msg("successfully unlocked PR")
		}
	}

	return unlockFn, nil
}
