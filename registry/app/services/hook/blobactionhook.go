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
	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

type BlobActionHook interface {
	Commit(
		ctx context.Context,
		sha1 digest.Digest,
		sha256 digest.Digest,
		sha512 digest.Digest,
		md5 digest.Digest,
		size int64,
		bucketKey string,
		req types.DriverRequest,
	) error

	EmitReadEvent(ctx context.Context, sha256 digest.Digest, req types.DriverRequest) error
}

type noOpBlobActionHook struct {
}

func (b *noOpBlobActionHook) EmitReadEvent(ctx context.Context, sha256 digest.Digest, req types.DriverRequest) error {
	log.Ctx(ctx).Info().Msgf("BlobActionHook called for data: %v sha256: %s", req, sha256)
	return nil
}

func (b *noOpBlobActionHook) Commit(
	ctx context.Context,
	sha1 digest.Digest,
	sha256 digest.Digest,
	sha512 digest.Digest,
	md5 digest.Digest,
	size int64,
	bucketKey string,
	req types.DriverRequest,
) error {
	log.Ctx(ctx).Info().Msgf("BlobActionHook called for data: %v sha1: %s sha256: %s "+
		"sha512: %s md5: %s size: %d bucketKey: %s",
		req, sha1, sha256, sha512, md5, size, bucketKey)
	return nil
}

func NewNoOpBlobActionHook() BlobActionHook {
	return &noOpBlobActionHook{}
}
