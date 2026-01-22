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

package python

import (
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/python"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

func ControllerProvider(
	proxyStore store.UpstreamProxyConfigRepository,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	urlProvider urlprovider.Provider,
	local python.LocalRegistry,
	proxy python.Proxy,
	quarantineFinder quarantine.Finder,
	dependencyFirewallChecker interfaces.DependencyFirewallChecker,
) Controller {
	return NewController(proxyStore, registryDao,
		imageDao, artifactDao, fileManager, tx, urlProvider, local, proxy, quarantineFinder, dependencyFirewallChecker)
}

var ControllerSet = wire.NewSet(ControllerProvider)
