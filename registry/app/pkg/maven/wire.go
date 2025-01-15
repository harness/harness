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
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

func LocalRegistryProvider(
	dBStore *DBStore,
	tx dbtx.Transactor,
) *LocalRegistry {
	return NewLocalRegistry(dBStore, tx).(*LocalRegistry)
}

func RemoteRegistryProvider(
	dBStore *DBStore,
	tx dbtx.Transactor,
) *RemoteRegistry {
	return NewRemoteRegistry(dBStore, tx).(*RemoteRegistry)
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
) *DBStore {
	return NewDBStore(registryDao, imageDao, artifactDao, spaceStore, bandwidthStatDao, downloadStatDao)
}

var ControllerSet = wire.NewSet(ControllerProvider)
var DBStoreSet = wire.NewSet(DBStoreProvider)
var RegistrySet = wire.NewSet(LocalRegistryProvider, RemoteRegistryProvider)
var WireSet = wire.NewSet(ControllerSet, DBStoreSet, RegistrySet)
