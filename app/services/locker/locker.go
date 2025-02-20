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

	"github.com/harness/gitness/contextutil"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const namespaceRepo = "repo"

type Locker struct {
	mtxManager lock.MutexManager
}

func NewLocker(mtxManager lock.MutexManager) *Locker {
	return &Locker{
		mtxManager: mtxManager,
	}
}

func (l Locker) lock(
	ctx context.Context,
	namespace string,
	key string,
	expiry time.Duration,
) (func(), error) {
	// annotate logs for easier debugging of lock related issues
	ctx = logging.NewContext(ctx, func(zc zerolog.Context) zerolog.Context {
		return zc.
			Str("key", key).
			Str("namespace", namespaceRepo).
			Str("expiry", expiry.String())
	})

	mutext, err := l.mtxManager.NewMutex(
		key,
		lock.WithNamespace(namespace),
		lock.WithExpiry(expiry),
		lock.WithTimeoutFactor(4/expiry.Seconds()), // 4s
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new mutex: %w", err)
	}

	log.Ctx(ctx).Debug().Msg("attempting to acquire lock")

	err = mutext.Lock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to lock the mutex: %w", err)
	}

	log.Ctx(ctx).Debug().Msgf("successfully locked (expiry: %s)", expiry)

	unlockFn := func() {
		// always unlock independent of whether source context got canceled or not
		ctx, cancel := contextutil.WithNewTimeout(ctx, 30*time.Second)
		defer cancel()

		err := mutext.Unlock(ctx)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to unlock")
		} else {
			log.Ctx(ctx).Debug().Msg("successfully unlocked")
		}
	}

	return unlockFn, nil
}
