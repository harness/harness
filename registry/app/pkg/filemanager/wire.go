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

package filemanager

import (
	"github.com/harness/gitness/registry/app/event"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/google/wire"
)

func Provider(
	registryDao store.RegistryRepository, genericBlobDao store.GenericBlobRepository,
	nodesDao store.NodesRepository,
	tx dbtx.Transactor,
	reporter event.Reporter,
	config *gitnesstypes.Config,
	storageService *storage.Service,
) FileManager {
	return NewFileManager(registryDao, genericBlobDao, nodesDao, tx, reporter, config, storageService)
}

var Set = wire.NewSet(Provider)

var WireSet = wire.NewSet(Set)
