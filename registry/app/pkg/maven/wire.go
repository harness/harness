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

package maven

import (
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/remote/controller/proxy/maven"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

func LocalRegistryProvider(
	dBStore *DBStore,
	tx dbtx.Transactor,
	fileManager filemanager.FileManager,
) *LocalRegistry {
	//nolint:errcheck
	return NewLocalRegistry(dBStore,
		tx,
		fileManager,
	).(*LocalRegistry)
}

func RemoteRegistryProvider(
	dBStore *DBStore,
	tx dbtx.Transactor,
	local *LocalRegistry,
	proxyController maven.Controller,
) *RemoteRegistry {
	//nolint:errcheck
	return NewRemoteRegistry(dBStore, tx, local, proxyController).(*RemoteRegistry)
}

func ControllerProvider(
	local *LocalRegistry,
	remote *RemoteRegistry,
	authorizer authz.Authorizer,
	dBStore *DBStore,
) *Controller {
	return NewController(local, remote, authorizer, dBStore)
}

func DBStoreProvider(
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	spaceStore corestore.SpaceStore,
	bandwidthStatDao store.BandwidthStatRepository,
	downloadStatDao store.DownloadStatRepository,
	nodeDao store.NodesRepository,
	upstreamProxyDao store.UpstreamProxyConfigRepository,
) *DBStore {
	//nolint:errcheck
	return NewDBStore(registryDao, imageDao, artifactDao, spaceStore, bandwidthStatDao,
		downloadStatDao,
		nodeDao,
		upstreamProxyDao)
}

func ProvideProxyController(
	registry *LocalRegistry, secretService secret.Service,
	spaceFinder refcache.SpaceFinder,
) maven.Controller {
	return maven.NewProxyController(registry, secretService, spaceFinder)
}

var ControllerSet = wire.NewSet(ControllerProvider)
var DBStoreSet = wire.NewSet(DBStoreProvider)
var RegistrySet = wire.NewSet(LocalRegistryProvider, RemoteRegistryProvider)
var ProxySet = wire.NewSet(ProvideProxyController)
var WireSet = wire.NewSet(ControllerSet, DBStoreSet, RegistrySet, ProxySet)
