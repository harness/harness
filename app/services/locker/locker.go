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
const namespaceRegistry = "registry"

type Locker struct {
	mtxManager lock.MutexManager
}

func NewLocker(mtxManager lock.MutexManager) *Locker {
	return &Locker{
		mtxManager: mtxManager,
	}
}

// defaultWait is the time a caller waits to acquire a lock unless overridden with WithWait.
const defaultWait = 4 * time.Second

// retryDelay is the pause between two attempts to acquire a lock when a custom wait time is used.
const retryDelay = 250 * time.Millisecond

// Option customizes how a lock is acquired.
type Option func(*options)

type options struct {
	wait time.Duration
}

// WithWait sets the maximum time to wait for a held lock to be released before giving up.
// Callers that can afford to queue (for example a merge on a busy branch) should use it
// instead of failing fast with a temporarily locked error.
func WithWait(wait time.Duration) Option {
	return func(o *options) {
		if wait > 0 {
			o.wait = wait
		}
	}
}

func (l Locker) lock(
	ctx context.Context,
	namespace string,
	key string,
	expiry time.Duration,
	opts ...Option,
) (func(), error) {
	o := options{wait: defaultWait}
	for _, opt := range opts {
		opt(&o)
	}

	// annotate logs for easier debugging of lock related issues
	ctx = logging.NewContext(ctx, func(zc zerolog.Context) zerolog.Context {
		return zc.
			Str("key", key).
			Str("namespace", namespace).
			Str("expiry", expiry.String())
	})

	lockOpts := []lock.Option{
		lock.WithNamespace(namespace),
		lock.WithExpiry(expiry),
		lock.WithTimeoutFactor(o.wait.Seconds() / expiry.Seconds()),
	}
	if o.wait != defaultWait {
		// the configured tries and retry delay only cover about 2s, so size them for the requested wait.
		lockOpts = append(lockOpts,
			lock.WithRetryDelay(retryDelay),
			lock.WithTries(int(o.wait/retryDelay)+1),
		)
	}

	mutext, err := l.mtxManager.NewMutex(key, lockOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create new mutex: %w", err)
	}

	log.Ctx(ctx).Debug().Msg("attempting to acquire lock")

	// the timeout factor bounds a single attempt only, so cap the whole acquisition at the requested wait.
	// The original ctx is kept for the logs and for unlocking.
	lockCtx, cancelLock := context.WithTimeout(ctx, o.wait)
	err = mutext.Lock(lockCtx)
	waitExpired := lockCtx.Err() != nil && ctx.Err() == nil
	cancelLock()
	if err != nil {
		if waitExpired {
			// report it like any other failure to get the lock (423) and not as an internal error.
			err = lock.NewError(lock.ErrorKindCannotLock, key, err)
		}
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
