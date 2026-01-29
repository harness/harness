//  Copyright 2023 Harness, Inc.
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

package hook

import (
	"context"
	"time"

	cfg "github.com/harness/gitness/registry/config"

	"github.com/rs/zerolog/log"
)

const (
	// DefaultAsyncTimeout is the default timeout for async hook operations.
	DefaultAsyncTimeout = 10 * time.Second
)

// EmitReadEventAsync emits a read event asynchronously in a goroutine.
// The event contains all necessary context for external tracking/billing.
func EmitReadEventAsync(
	ctx context.Context,
	hook BlobActionHook,
	event BlobReadEvent,
) {
	go func() {
		ctx2 := context.WithoutCancel(ctx)
		ctx2, cancel := context.WithTimeout(ctx2, DefaultAsyncTimeout)
		defer cancel()
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "Emit Blob Read Event")

		if err := hook.OnRead(ctx2, event); err != nil {
			log.Ctx(ctx2).Error().Err(err).
				Str("digest", event.BlobLocator.Digest.String()).
				Msg("failed to emit read event")
		}
	}()
}
