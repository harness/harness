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

package generic

import (
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	gitnessstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/generic"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

func DBStoreProvider(
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	bandwidthStatDao store.BandwidthStatRepository,
	downloadStatDao store.DownloadStatRepository,
	registryDao store.RegistryRepository,
) *DBStore {
	return NewDBStore(registryDao, imageDao, artifactDao, bandwidthStatDao, downloadStatDao)
}

func ControllerProvider(
	spaceStore gitnessstore.SpaceStore,
	authorizer authz.Authorizer,
	fileManager filemanager.FileManager,
	dBStore *DBStore,
	tx dbtx.Transactor,
	spaceFinder refcache.SpaceFinder,
	local generic.LocalRegistry,
	proxy generic.Proxy,
	quarantineFinder quarantine.Finder,
	dependencyFirewallChecker interfaces.DependencyFirewallChecker,
	auditService audit.Service,
) *Controller {
	return NewController(
		spaceStore,
		authorizer,
		fileManager,
		dBStore,
		tx,
		spaceFinder,
		local,
		proxy,
		quarantineFinder,
		dependencyFirewallChecker,
		auditService,
	)
}

var DBStoreSet = wire.NewSet(DBStoreProvider)
var ControllerSet = wire.NewSet(ControllerProvider)

var WireSet = wire.NewSet(ControllerSet, DBStoreSet)
