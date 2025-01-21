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

package metadata

import (
	"github.com/harness/gitness/app/auth/authz"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

// APIController simple struct.
type APIController struct {
	ImageStore         store.ImageRepository
	RegistryRepository store.RegistryRepository
	UpstreamProxyStore store.UpstreamProxyConfigRepository
	TagStore           store.TagRepository
	ManifestStore      store.ManifestRepository
	CleanupPolicyStore store.CleanupPolicyRepository
	SpaceStore         corestore.SpaceStore
	tx                 dbtx.Transactor
	StorageDriver      storagedriver.StorageDriver
	URLProvider        urlprovider.Provider
	Authorizer         authz.Authorizer
	AuditService       audit.Service
	spacePathStore     corestore.SpacePathStore
	ArtifactStore      store.ArtifactRepository
}

func NewAPIController(
	repositoryStore store.RegistryRepository,
	upstreamProxyStore store.UpstreamProxyConfigRepository,
	tagStore store.TagRepository,
	manifestStore store.ManifestRepository,
	cleanupPolicyStore store.CleanupPolicyRepository,
	imageStore store.ImageRepository,
	driver storagedriver.StorageDriver,
	spaceStore corestore.SpaceStore,
	tx dbtx.Transactor,
	urlProvider urlprovider.Provider,
	authorizer authz.Authorizer,
	auditService audit.Service,
	spacePathStore corestore.SpacePathStore,
	artifactStore store.ArtifactRepository,
) *APIController {
	return &APIController{
		RegistryRepository: repositoryStore,
		UpstreamProxyStore: upstreamProxyStore,
		TagStore:           tagStore,
		ManifestStore:      manifestStore,
		CleanupPolicyStore: cleanupPolicyStore,
		ImageStore:         imageStore,
		SpaceStore:         spaceStore,
		StorageDriver:      driver,
		tx:                 tx,
		URLProvider:        urlProvider,
		Authorizer:         authorizer,
		AuditService:       auditService,
		spacePathStore:     spacePathStore,
		ArtifactStore:      artifactStore,
	}
}
