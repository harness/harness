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
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

type BucketKeyGetter interface {
	GetBucketKey() string
}

// EmitReadEventAsync emits a read event asynchronously in a goroutine.
func EmitReadEventAsync(
	ctx context.Context,
	blobActionHook BlobActionHook,
	rootIdentifier string,
	sha256Digest types.Digest,
) {
	go func() {
		ctx2 := context.WithoutCancel(ctx)
		ctx2, cancel := context.WithTimeout(ctx2, 10*time.Second)
		defer cancel()
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "Emit Generic Read Event")
		err := blobActionHook.EmitReadEvent(ctx2, rootIdentifier, sha256Digest)
		if err != nil {
			log.Ctx(ctx2).Error().Err(err).Msgf("Failed to emit Read Event for digest: %s", sha256Digest)
		}
	}()
}

func GetBucketKey(driverDetails BucketKeyGetter) string {
	if driverDetails == nil {
		return ""
	}
	return driverDetails.GetBucketKey()
}
