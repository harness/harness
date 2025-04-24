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
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/interfaces"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	registryevents "github.com/harness/gitness/registry/app/events"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/services/index"
	"github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/store/database/dbtx"
)

// APIController simple struct.
type APIController struct {
	ImageStore                  store.ImageRepository
	fileManager                 filemanager.FileManager
	BlobStore                   store.BlobRepository
	GenericBlobStore            store.GenericBlobRepository
	RegistryRepository          store.RegistryRepository
	UpstreamProxyStore          store.UpstreamProxyConfigRepository
	TagStore                    store.TagRepository
	ManifestStore               store.ManifestRepository
	CleanupPolicyStore          store.CleanupPolicyRepository
	SpaceFinder                 interfaces.SpaceFinder
	tx                          dbtx.Transactor
	StorageDriver               storagedriver.StorageDriver
	URLProvider                 urlprovider.Provider
	Authorizer                  authz.Authorizer
	AuditService                audit.Service
	ArtifactStore               store.ArtifactRepository
	WebhooksRepository          store.WebhooksRepository
	WebhooksExecutionRepository store.WebhooksExecutionRepository
	RegistryMetadataHelper      interfaces.RegistryMetadataHelper
	WebhookService              webhook.ServiceInterface
	ArtifactEventReporter       registryevents.Reporter
	DownloadStatRepository      store.DownloadStatRepository
	RegistryIndexService        index.Service
}

func NewAPIController(
	repositoryStore store.RegistryRepository,
	fileManager filemanager.FileManager,
	blobStore store.BlobRepository,
	genericBlobStore store.GenericBlobRepository,
	upstreamProxyStore store.UpstreamProxyConfigRepository,
	tagStore store.TagRepository,
	manifestStore store.ManifestRepository,
	cleanupPolicyStore store.CleanupPolicyRepository,
	imageStore store.ImageRepository,
	driver storagedriver.StorageDriver,
	spaceFinder interfaces.SpaceFinder,
	tx dbtx.Transactor,
	urlProvider urlprovider.Provider,
	authorizer authz.Authorizer,
	auditService audit.Service,
	artifactStore store.ArtifactRepository,
	webhooksRepository store.WebhooksRepository,
	webhooksExecutionRepository store.WebhooksExecutionRepository,
	registryMetadataHelper interfaces.RegistryMetadataHelper,
	webhookService webhook.ServiceInterface,
	artifactEventReporter registryevents.Reporter,
	downloadStatRepository store.DownloadStatRepository,
	registryIndexService index.Service,
) *APIController {
	return &APIController{
		fileManager:                 fileManager,
		GenericBlobStore:            genericBlobStore,
		BlobStore:                   blobStore,
		RegistryRepository:          repositoryStore,
		UpstreamProxyStore:          upstreamProxyStore,
		TagStore:                    tagStore,
		ManifestStore:               manifestStore,
		CleanupPolicyStore:          cleanupPolicyStore,
		ImageStore:                  imageStore,
		SpaceFinder:                 spaceFinder,
		StorageDriver:               driver,
		tx:                          tx,
		URLProvider:                 urlProvider,
		Authorizer:                  authorizer,
		AuditService:                auditService,
		ArtifactStore:               artifactStore,
		WebhooksRepository:          webhooksRepository,
		WebhooksExecutionRepository: webhooksExecutionRepository,
		RegistryMetadataHelper:      registryMetadataHelper,
		WebhookService:              webhookService,
		ArtifactEventReporter:       artifactEventReporter,
		DownloadStatRepository:      downloadStatRepository,
		RegistryIndexService:        registryIndexService,
	}
}
