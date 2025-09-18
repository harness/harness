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
	"context"

	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	gitnessstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/event"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg"
	proxy2 "github.com/harness/gitness/registry/app/remote/controller/proxy"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/gc"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func LocalRegistryProvider(
	app *App, ms ManifestService, blobRepo store.BlobRepository,
	registryDao store.RegistryRepository, manifestDao store.ManifestRepository,
	registryBlobDao store.RegistryBlobRepository,
	mtRepository store.MediaTypesRepository,
	tagDao store.TagRepository, imageDao store.ImageRepository, artifactDao store.ArtifactRepository,
	bandwidthStatDao store.BandwidthStatRepository, downloadStatDao store.DownloadStatRepository,
	gcService gc.Service, tx dbtx.Transactor, reporter event.Reporter,
	quarantineArtifactDao store.QuarantineArtifactRepository,
	bucketService BucketService,
) *LocalRegistry {
	registry, ok := NewLocalRegistry(
		app, ms, manifestDao, registryDao, registryBlobDao, blobRepo,
		mtRepository, tagDao, imageDao, artifactDao, bandwidthStatDao, downloadStatDao,
		gcService, tx, reporter, quarantineArtifactDao, bucketService,
	).(*LocalRegistry)
	if !ok {
		return nil
	}
	return registry
}

func ManifestServiceProvider(
	registryDao store.RegistryRepository,
	manifestDao store.ManifestRepository, blobRepo store.BlobRepository, mtRepository store.MediaTypesRepository,
	manifestRefDao store.ManifestReferenceRepository, tagDao store.TagRepository, imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository, layerDao store.LayerRepository,
	gcService gc.Service, tx dbtx.Transactor, reporter event.Reporter, spaceFinder refcache.SpaceFinder,
	ociImageIndexMappingDao store.OCIImageIndexMappingRepository,
	artifactEventReporter *registryevents.Reporter,
	urlProvider url.Provider,
) ManifestService {
	return NewManifestService(
		registryDao, manifestDao, blobRepo, mtRepository, tagDao, imageDao,
		artifactDao, layerDao, manifestRefDao, tx, gcService, reporter, spaceFinder,
		ociImageIndexMappingDao, *artifactEventReporter, urlProvider,
	)
}

func RemoteRegistryProvider(
	local *LocalRegistry, app *App, upstreamProxyConfigRepo store.UpstreamProxyConfigRepository,
	spaceFinder refcache.SpaceFinder, secretService secret.Service, proxyCtrl proxy2.Controller,
) *RemoteRegistry {
	registry, ok := NewRemoteRegistry(local, app, upstreamProxyConfigRepo, spaceFinder, secretService,
		proxyCtrl).(*RemoteRegistry)
	if !ok {
		return nil
	}
	return registry
}

func ControllerProvider(
	local *LocalRegistry,
	remote *RemoteRegistry,
	controller *pkg.CoreController,
	spaceStore gitnessstore.SpaceStore,
	authorizer authz.Authorizer,
	dBStore *DBStore,
	spaceFinder refcache.SpaceFinder,
) *Controller {
	return NewController(local, remote, controller, spaceStore, authorizer, dBStore, spaceFinder)
}

func DBStoreProvider(
	blobRepo store.BlobRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	bandwidthStatDao store.BandwidthStatRepository,
	downloadStatDao store.DownloadStatRepository,
	manifestDao store.ManifestRepository,
	quarantineDao store.QuarantineArtifactRepository,
) *DBStore {
	return NewDBStore(blobRepo, imageDao, artifactDao, bandwidthStatDao, downloadStatDao, manifestDao, quarantineDao)
}

func StorageServiceProvider(cfg *types.Config, driver storagedriver.StorageDriver) *storage.Service {
	return GetStorageService(cfg, driver)
}

func ProvideReporter() event.Reporter {
	return &event.Noop{}
}

func ProvideProxyController(
	registry *LocalRegistry, ms ManifestService, secretService secret.Service,
	spaceFinder refcache.SpaceFinder,
) proxy2.Controller {
	manifestCacheHandler := getManifestCacheHandler(registry, ms)
	return proxy2.NewProxyController(registry, ms, secretService, spaceFinder, manifestCacheHandler)
}

func getManifestCacheHandler(
	registry *LocalRegistry, ms ManifestService,
) map[string]proxy2.ManifestCacheHandler {
	cache := proxy2.GetManifestCache(registry, ms)
	listCache := proxy2.GetManifestListCache(registry)

	return map[string]proxy2.ManifestCacheHandler{
		manifestlist.MediaTypeManifestList: listCache,
		v1.MediaTypeImageIndex:             listCache,
		schema2.MediaTypeManifest:          cache,
		proxy2.DefaultHandler:              cache,
	}
}

var ControllerSet = wire.NewSet(ControllerProvider)
var DBStoreSet = wire.NewSet(DBStoreProvider)
var RegistrySet = wire.NewSet(LocalRegistryProvider, ManifestServiceProvider, RemoteRegistryProvider)
var ProxySet = wire.NewSet(ProvideProxyController)
var StorageServiceSet = wire.NewSet(StorageServiceProvider)
var AppSet = wire.NewSet(NewApp)

// OciBlobStoreFactory is a function that creates an OciBlobStore with the provided context and identifiers.
type OciBlobStoreFactory func(ctx context.Context, repoKey string, rootParentRef string) storage.OciBlobStore

// ProvideOciBlobStore returns a factory function that creates an OciBlobStore with the provided context.
func ProvideOciBlobStore(storageService *storage.Service) OciBlobStoreFactory {
	return storageService.OciBlobsStore
}

func ProvideBucketService(_ OciBlobStoreFactory) BucketService {
	return &noOpBucketService{}
}

// noOpBucketService is a no-op implementation for open-source version.
type noOpBucketService struct{}

func (n *noOpBucketService) GetBlobStore(_ context.Context, _ string, _ string,
	_ interface{}, _ string) *BlobStore {
	return nil
}

var OciBlobStoreSet = wire.NewSet(ProvideOciBlobStore)
var BucketServiceSet = wire.NewSet(ProvideBucketService)

// WireSet is used without the OciBlobStore and BucketService.
var WireSet = wire.NewSet(
	ControllerSet,
	DBStoreSet,
	RegistrySet,
	StorageServiceSet,
	AppSet,
	ProxySet,
)

// OpenSourceWireSet is used by the open-source version.
var OpenSourceWireSet = wire.NewSet(
	WireSet,
	OciBlobStoreSet,
	BucketServiceSet,
)
