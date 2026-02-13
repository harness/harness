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

	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

type BlobEventBase struct {
	BlobLocator types.BlobLocator
	ClientIP    string
	BucketKey   types.BucketKey
}

type BlobCommitEvent struct {
	BlobEventBase
	Digests types.BlobDigests
	Size    int64
}

type BlobReadEvent struct {
	BlobEventBase
}

type BlobActionHook interface {
	// OnCommit is called when a blob is successfully committed to storage.
	OnCommit(ctx context.Context, event BlobCommitEvent) error

	// OnRead is called when a blob is read.
	OnRead(ctx context.Context, event BlobReadEvent) error
}

type noOpBlobActionHook struct{}

func (b *noOpBlobActionHook) OnRead(ctx context.Context, event BlobReadEvent) error {
	log.Ctx(ctx).Trace().
		Str("sha256", event.BlobLocator.Digest.String()).
		Int64("registry_id", event.BlobLocator.RegistryID).
		Int64("root_parent_id", event.BlobLocator.RootParentID).
		Str("bucket_key", string(event.BucketKey)).
		Msg("BlobActionHook.OnRead called")
	return nil
}

func (b *noOpBlobActionHook) OnCommit(ctx context.Context, event BlobCommitEvent) error {
	log.Ctx(ctx).Trace().
		Str("sha1", event.Digests.SHA1.String()).
		Str("sha256", event.Digests.SHA256.String()).
		Str("sha512", event.Digests.SHA512.String()).
		Str("md5", event.Digests.MD5.String()).
		Int64("size", event.Size).
		Str("bucket_key", string(event.BucketKey)).
		Int64("registry_id", event.BlobLocator.RegistryID).
		Int64("root_parent_id", event.BlobLocator.RootParentID).
		Msg("BlobActionHook.OnCommit called")
	return nil
}

func NewNoOpBlobActionHook() BlobActionHook {
	return &noOpBlobActionHook{}
}
