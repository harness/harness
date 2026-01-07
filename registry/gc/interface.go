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

package gc

import (
	"context"
	"time"

	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"
)

type Service interface {
	Start(
		ctx context.Context,
		spaceStore corestore.SpaceStore,
		blobRepo store.BlobRepository,
		storageDeleter *storage.Service,
		config *types.Config,
	)
	BlobFindAndLockBefore(ctx context.Context, blobID int64, date time.Time) (*registrytypes.GCBlobTask, error)
	BlobReschedule(ctx context.Context, b *registrytypes.GCBlobTask, d time.Duration) error
	ManifestFindAndLockBefore(
		ctx context.Context, registryID, manifestID int64,
		date time.Time,
	) (*registrytypes.GCManifestTask, error)
	ManifestFindAndLockNBefore(
		ctx context.Context, registryID int64, manifestIDs []int64,
		date time.Time,
	) ([]*registrytypes.GCManifestTask, error)
}
