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

package base

import (
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

func LocalBaseProvider(
	registryDao store.RegistryRepository,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	nodesDao store.NodesRepository,
	tagsDao store.PackageTagRepository,
	authorizer authz.Authorizer,
	spaceFinder refcache.SpaceFinder,
) LocalBase {
	return NewLocalBase(registryDao, fileManager, tx, imageDao, artifactDao, nodesDao, tagsDao, authorizer, spaceFinder)
}

var WireSet = wire.NewSet(LocalBaseProvider)
