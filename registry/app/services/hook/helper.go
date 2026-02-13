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

	"github.com/rs/zerolog/log"
)

// EmitReadEvent emits a read event.
// Important: The implementer should trigger this in async as this is on active read path and it is responsibility
// of the implementer of this hook.
func EmitReadEvent(
	ctx context.Context,
	hook BlobActionHook,
	event BlobReadEvent,
) {
	if err := hook.OnRead(ctx, event); err != nil {
		log.Ctx(ctx).Error().Err(err).
			Str("digest", event.BlobLocator.Digest.String()).
			Msg("failed to emit read event")
	}
}

// EmitCommitEventCallback returns a callback function that emits a commit event when executed.
// The callback should be executed inside a database transaction so that if it fails,
// the transaction can be rolled back.
func EmitCommitEventCallback(
	ctx context.Context,
	hook BlobActionHook,
	event BlobCommitEvent,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if err := hook.OnCommit(ctx, event); err != nil {
			log.Ctx(ctx).Error().Err(err).
				Str("sha256", event.Digests.SHA256.String()).
				Int64("size", event.Size).
				Str("bucket_key", string(event.BucketKey)).
				Int64("registry_id", event.BlobLocator.RegistryID).
				Int64("root_parent_id", event.BlobLocator.RootParentID).
				Msg("failed to emit commit event")
			return err
		}
		return nil
	}
}
