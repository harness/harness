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

package docker

import (
	"github.com/harness/gitness/app/auth/authz"
	corestore "github.com/harness/gitness/app/store"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/gc"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

func LocalRegistryProvider(
	app *App, ms ManifestService, blobRepo store.BlobRepository,
	registryDao store.RegistryRepository, manifestDao store.ManifestRepository,
	registryBlobDao store.RegistryBlobRepository,
	mtRepository store.MediaTypesRepository,
	tagDao store.TagRepository, artifactDao store.ArtifactRepository, artifactStatDao store.ArtifactStatRepository,
	gcService gc.Service, tx dbtx.Transactor,
) *LocalRegistry {
	return NewLocalRegistry(
		app, ms, manifestDao, registryDao, registryBlobDao, blobRepo,
		mtRepository, tagDao, artifactDao, artifactStatDao, gcService, tx,
	).(*LocalRegistry)
}

func ManifestServiceProvider(
	registryDao store.RegistryRepository,
	manifestDao store.ManifestRepository, blobRepo store.BlobRepository, mtRepository store.MediaTypesRepository,
	manifestRefDao store.ManifestReferenceRepository, tagDao store.TagRepository, artifactDao store.ArtifactRepository,
	artifactStatDao store.ArtifactStatRepository, layerDao store.LayerRepository,
	gcService gc.Service, tx dbtx.Transactor,
) ManifestService {
	return NewManifestService(
		registryDao, manifestDao, blobRepo, mtRepository, tagDao,
		artifactDao, artifactStatDao, layerDao, manifestRefDao, tx, gcService,
	)
}

func RemoteRegistryProvider(
	local *LocalRegistry, app *App, upstreamProxyConfigRepo store.UpstreamProxyConfigRepository,
	spacePathStore corestore.SpacePathStore, secretService secret.Service,
) *RemoteRegistry {
	return NewRemoteRegistry(local, app, upstreamProxyConfigRepo, spacePathStore, secretService).(*RemoteRegistry)
}

func ControllerProvider(
	local *LocalRegistry,
	remote *RemoteRegistry,
	controller *pkg.CoreController,
	spaceStore corestore.SpaceStore,
	authorizer authz.Authorizer,
) *Controller {
	return NewController(local, remote, controller, spaceStore, authorizer)
}

func StorageServiceProvider(cfg *types.Config, driver storagedriver.StorageDriver) *storage.Service {
	return GetStorageService(cfg, driver)
}

var ControllerSet = wire.NewSet(ControllerProvider)
var RegistrySet = wire.NewSet(LocalRegistryProvider, ManifestServiceProvider, RemoteRegistryProvider)
var StorageServiceSet = wire.NewSet(StorageServiceProvider)
var AppSet = wire.NewSet(NewApp)
var WireSet = wire.NewSet(ControllerSet, RegistrySet, StorageServiceSet, AppSet, gc.WireSet)
