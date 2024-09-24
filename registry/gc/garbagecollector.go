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
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

type Noop struct{}

func New() Service {
	return &Noop{}
}

func (s *Noop) Start(
	_ context.Context, _ *sqlx.DB, _ corestore.SpaceStore,
	_ store.BlobRepository, _ storagedriver.StorageDeleter,
	_ *types.Config,
) {
	// NOOP
}

func (s *Noop) BlobFindAndLockBefore(_ context.Context, _ int64, _ time.Time) (*registrytypes.GCBlobTask, error) {
	// NOOP
	//nolint:nilnil
	return nil, nil
}

func (s *Noop) BlobReschedule(_ context.Context, _ *registrytypes.GCBlobTask, _ time.Duration) error {
	// NOOP
	return nil
}

func (s *Noop) ManifestFindAndLockBefore(_ context.Context, _, _ int64, _ time.Time) (
	*registrytypes.GCManifestTask, error,
) {
	// NOOP
	//nolint:nilnil
	return nil, nil
}

func (s *Noop) ManifestFindAndLockNBefore(_ context.Context, _ int64, _ []int64, _ time.Time) (
	[]*registrytypes.GCManifestTask, error,
) {
	// NOOP
	return nil, nil
}
