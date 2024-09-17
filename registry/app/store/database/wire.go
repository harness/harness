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

package database

import (
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

func ProvideUpstreamDao(
	db *sqlx.DB,
	registryDao store.RegistryRepository,
	spacePathStore corestore.SpacePathStore,
) store.UpstreamProxyConfigRepository {
	return NewUpstreamproxyDao(db, registryDao, spacePathStore)
}

func ProvideRepoDao(db *sqlx.DB, mtRepository store.MediaTypesRepository) store.RegistryRepository {
	return NewRegistryDao(db, mtRepository)
}

func ProvideMediaTypeDao(db *sqlx.DB) store.MediaTypesRepository {
	return NewMediaTypesDao(db)
}

func ProvideBlobDao(db *sqlx.DB, mtRepository store.MediaTypesRepository) store.BlobRepository {
	return NewBlobDao(db, mtRepository)
}

func ProvideRegistryBlobDao(db *sqlx.DB) store.RegistryBlobRepository {
	return NewRegistryBlobDao(db)
}

func ProvideImageDao(db *sqlx.DB) store.ImageRepository {
	return NewImageDao(db)
}

func ProvideArtifactDao(db *sqlx.DB) store.ArtifactRepository {
	return NewArtifactDao(db)
}

func ProvideDownloadStatDao(db *sqlx.DB) store.DownloadStatRepository {
	return NewDownloadStatDao(db)
}

func ProvideBandwidthStatDao(db *sqlx.DB) store.BandwidthStatRepository {
	return NewBandwidthStatDao(db)
}

func ProvideTagDao(db *sqlx.DB) store.TagRepository {
	return NewTagDao(db)
}

func ProvideManifestDao(sqlDB *sqlx.DB, mtRepository store.MediaTypesRepository) store.ManifestRepository {
	return NewManifestDao(sqlDB, mtRepository)
}

func ProvideManifestRefDao(db *sqlx.DB) store.ManifestReferenceRepository {
	return NewManifestReferenceDao(db)
}

func ProvideLayerDao(db *sqlx.DB, mtRepository store.MediaTypesRepository) store.LayerRepository {
	return NewLayersDao(db, mtRepository)
}

func ProvideCleanupPolicyDao(db *sqlx.DB, tx dbtx.Transactor) store.CleanupPolicyRepository {
	return NewCleanupPolicyDao(db, tx)
}

var WireSet = wire.NewSet(
	ProvideUpstreamDao,
	ProvideRepoDao,
	ProvideMediaTypeDao,
	ProvideBlobDao,
	ProvideRegistryBlobDao,
	ProvideTagDao,
	ProvideManifestDao,
	ProvideCleanupPolicyDao,
	ProvideManifestRefDao,
	ProvideLayerDao,
	ProvideImageDao,
	ProvideArtifactDao,
	ProvideDownloadStatDao,
	ProvideBandwidthStatDao,
)
