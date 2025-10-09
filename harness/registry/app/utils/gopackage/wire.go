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

package gopackage

import (
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/store"

	"github.com/google/wire"
)

func LocalRegistryHelperProvider(
	fileManager filemanager.FileManager,
	artifactDao store.ArtifactRepository,
	spaceFinder refcache.SpaceFinder,
	registryFinder refcache2.RegistryFinder,
) RegistryHelper {
	return NewRegistryHelper(fileManager, artifactDao, spaceFinder, registryFinder)
}

var WireSet = wire.NewSet(LocalRegistryHelperProvider)
