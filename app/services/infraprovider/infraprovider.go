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
)

func NewService(
	tx dbtx.Transactor,
	gitspaceConfigStore store.GitspaceConfigStore,
	resourceStore store.InfraProviderResourceStore,
	configStore store.InfraProviderConfigStore,
	templateStore store.InfraProviderTemplateStore,
	factory infraprovider.Factory,
	spaceFinder refcache.SpaceFinder,
) *Service {
	return &Service{
		tx:                         tx,
		infraProviderResourceStore: resourceStore,
		infraProviderConfigStore:   configStore,
		infraProviderTemplateStore: templateStore,
		infraProviderFactory:       factory,
		spaceFinder:                spaceFinder,
		gitspaceConfigStore:        gitspaceConfigStore,
	}
}

type Service struct {
	tx                         dbtx.Transactor
	gitspaceConfigStore        store.GitspaceConfigStore
	infraProviderResourceStore store.InfraProviderResourceStore
	infraProviderConfigStore   store.InfraProviderConfigStore
	infraProviderTemplateStore store.InfraProviderTemplateStore
	infraProviderFactory       infraprovider.Factory
	spaceFinder                refcache.SpaceFinder
}
