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

package docker

import (
	"context"

	"github.com/harness/gitness/registry/app/storage"
)

// BlobStore represents the result of a blob store lookup.
type BlobStore struct {
	OciStore     storage.OciBlobStore
	GenericStore storage.GenericBlobStore
}

// BucketService defines a unified interface for multi-bucket blob serving functionality.
// It supports both OCI registry blobs and generic file blobs with geolocation-based routing.
type BucketService interface {
	// GetBlobStore returns the appropriate blob store for the closest bucket based on geolocation.
	// For OCI operations, pass repoKey. For generic operations, pass empty string for repoKey.
	// blobID can be int64 for OCI blobs or string for generic blobs.
	// If no suitable bucket is found, returns nil and the caller should fall back to using their default blob store.
	GetBlobStore(ctx context.Context, repoKey string, rootIdentifier string,
		blobID any, digest string) *BlobStore
}
