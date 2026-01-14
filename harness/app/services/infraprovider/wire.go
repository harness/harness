// Copyright 2023 Harness, Inc.
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

package infraprovider

import (
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideInfraProvider,
)

func ProvideInfraProvider(
	tx dbtx.Transactor,
	gitspaceConfigStore store.GitspaceConfigStore,
	resourceStore store.InfraProviderResourceStore,
	configStore store.InfraProviderConfigStore,
	templateStore store.InfraProviderTemplateStore,
	infraProviderFactory infraprovider.Factory,
	spaceFinder refcache.SpaceFinder,
	gatewayStore store.CDEGatewayStore,
) *Service {
	return NewService(tx, gitspaceConfigStore, resourceStore, configStore, templateStore, infraProviderFactory,
		spaceFinder, gatewayStore)
}
