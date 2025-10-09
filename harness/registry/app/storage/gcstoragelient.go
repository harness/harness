// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"

	"github.com/harness/gitness/registry/app/driver"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

type GcStorageClient struct {
	StorageDeleter driver.StorageDeleter
}

func NewGcStorageClient(storageDeleter driver.StorageDeleter) *GcStorageClient {
	return &GcStorageClient{
		StorageDeleter: storageDeleter,
	}
}

// RemoveBlob removes a blob from the filesystem.
func (sc *GcStorageClient) RemoveBlob(ctx context.Context, dgst digest.Digest, rootParentRef string) error {
	blobPath, err := pathFor(blobPathSpec{digest: dgst, path: rootParentRef})
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("deleting blob from storage, digest: %s , path: %s", dgst.String(), rootParentRef)
	if err := sc.StorageDeleter.Delete(ctx, blobPath); err != nil {
		return err
	}

	return nil
}
