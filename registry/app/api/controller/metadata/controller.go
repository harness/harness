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
	"context"

	spacecontroller "github.com/harness/gitness/app/api/controller/space"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/publicaccess"
	gstore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/interfaces"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	"github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/utils/cargo"
	webhook "github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/udp"
)

var errPublicArtifactRegistryCreationDisabled = usererror.BadRequest("Public artifact registry creation is disabled.")
var errPublicAccessToArtifactRegistriesDisabled = usererror.BadRequest(
	"Public access to artifact registries is disabled.",
)

// APIController simple struct.
type APIController struct {
	ImageStore                   store.ImageRepository
	fileManager                  filemanager.FileManager
	BlobStore                    store.BlobRepository
	GenericBlobStore             store.GenericBlobRepository
	RegistryRepository           store.RegistryRepository
	UpstreamProxyStore           store.UpstreamProxyConfigRepository
	TagStore                     store.TagRepository
	ManifestStore                store.ManifestRepository
	CleanupPolicyStore           store.CleanupPolicyRepository
	SpaceFinder                  interfaces.SpaceFinder
	tx                           dbtx.Transactor
	db                           dbtx.Accessor
	StorageDriver                storagedriver.StorageDriver
	URLProvider                  urlprovider.Provider
	Authorizer                   authz.Authorizer
	AuditService                 audit.Service
	UDPService                   udp.Service
	ArtifactStore                store.ArtifactRepository
	WebhooksRepository           store.WebhooksRepository
	WebhooksExecutionRepository  store.WebhooksExecutionRepository
	RegistryMetadataHelper       interfaces.RegistryMetadataHelper
	WebhookService               webhook.ServiceInterface
	ArtifactEventReporter        registryevents.Reporter
	DownloadStatRepository       store.DownloadStatRepository
	SetupDetailsAuthHeaderPrefix string
	RegistryBlobStore            store.RegistryBlobRepository
	RegFinder                    refcache.RegistryFinder
	PostProcessingReporter       *registrypostprocessingevents.Reporter
	CargoRegistryHelper          cargo.RegistryHelper
	SpaceController              *spacecontroller.Controller
	QuarantineArtifactRepository store.QuarantineArtifactRepository
	QuarantineFinder             quarantine.Finder
	SpaceStore                   gstore.SpaceStore
	UntaggedImagesEnabled        func(ctx context.Context) bool
	PackageWrapper               interfaces.PackageWrapper
	PublicAccess                 publicaccess.Service
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
	db dbtx.Accessor,
	urlProvider urlprovider.Provider,
	authorizer authz.Authorizer,
	auditService audit.Service,
	udpService udp.Service,
	artifactStore store.ArtifactRepository,
	webhooksRepository store.WebhooksRepository,
	webhooksExecutionRepository store.WebhooksExecutionRepository,
	registryMetadataHelper interfaces.RegistryMetadataHelper,
	webhookService webhook.ServiceInterface,
	artifactEventReporter registryevents.Reporter,
	downloadStatRepository store.DownloadStatRepository,
	setupDetailsAuthHeaderPrefix string,
	registryBlobStore store.RegistryBlobRepository,
	regFinder refcache.RegistryFinder,
	postProcessingReporter *registrypostprocessingevents.Reporter,
	cargoRegistryHelper cargo.RegistryHelper,
	spaceController *spacecontroller.Controller,
	quarantineArtifactRepository store.QuarantineArtifactRepository,
	quarantineFinder quarantine.Finder,
	spaceStore gstore.SpaceStore,
	untaggedImagesEnabled func(ctx context.Context) bool,
	packageWrapper interfaces.PackageWrapper,
	publicAccess publicaccess.Service,
) *APIController {
	return &APIController{
		fileManager:                  fileManager,
		GenericBlobStore:             genericBlobStore,
		BlobStore:                    blobStore,
		RegistryRepository:           repositoryStore,
		UpstreamProxyStore:           upstreamProxyStore,
		TagStore:                     tagStore,
		ManifestStore:                manifestStore,
		CleanupPolicyStore:           cleanupPolicyStore,
		ImageStore:                   imageStore,
		SpaceFinder:                  spaceFinder,
		StorageDriver:                driver,
		tx:                           tx,
		db:                           db,
		URLProvider:                  urlProvider,
		Authorizer:                   authorizer,
		AuditService:                 auditService,
		UDPService:                   udpService,
		ArtifactStore:                artifactStore,
		WebhooksRepository:           webhooksRepository,
		WebhooksExecutionRepository:  webhooksExecutionRepository,
		RegistryMetadataHelper:       registryMetadataHelper,
		WebhookService:               webhookService,
		ArtifactEventReporter:        artifactEventReporter,
		DownloadStatRepository:       downloadStatRepository,
		SetupDetailsAuthHeaderPrefix: setupDetailsAuthHeaderPrefix,
		RegistryBlobStore:            registryBlobStore,
		RegFinder:                    regFinder,
		PostProcessingReporter:       postProcessingReporter,
		CargoRegistryHelper:          cargoRegistryHelper,
		SpaceController:              spaceController,
		QuarantineArtifactRepository: quarantineArtifactRepository,
		QuarantineFinder:             quarantineFinder,
		SpaceStore:                   spaceStore,
		UntaggedImagesEnabled:        untaggedImagesEnabled,
		PackageWrapper:               packageWrapper,
		PublicAccess:                 publicAccess,
	}
}
