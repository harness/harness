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

package router

import (
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/config"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/handler/cargo"
	"github.com/harness/gitness/registry/app/api/handler/generic"
	"github.com/harness/gitness/registry/app/api/handler/maven"
	"github.com/harness/gitness/registry/app/api/handler/npm"
	"github.com/harness/gitness/registry/app/api/handler/nuget"
	hoci "github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/handler/python"
	"github.com/harness/gitness/registry/app/api/handler/rpm"
	generic2 "github.com/harness/gitness/registry/app/api/router/generic"
	"github.com/harness/gitness/registry/app/api/router/harness"
	mavenRouter "github.com/harness/gitness/registry/app/api/router/maven"
	"github.com/harness/gitness/registry/app/api/router/oci"
	packagerrouter "github.com/harness/gitness/registry/app/api/router/packages"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	registrywebhook "github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

func AppRouterProvider(
	ocir oci.RegistryOCIHandler,
	appHandler harness.APIHandler,
	mavenHandler mavenRouter.Handler,
	genericHandler generic2.Handler,
	handler packagerrouter.Handler,
) AppRouter {
	return GetAppRouter(ocir, appHandler, config.APIURL, mavenHandler, genericHandler, handler)
}

func APIHandlerProvider(
	repoDao store.RegistryRepository,
	upstreamproxyDao store.UpstreamProxyConfigRepository,
	fileManager filemanager.FileManager,
	tagDao store.TagRepository,
	manifestDao store.ManifestRepository,
	cleanupPolicyDao store.CleanupPolicyRepository,
	imageDao store.ImageRepository,
	driver storagedriver.StorageDriver,
	spaceFinder refcache.SpaceFinder,
	tx dbtx.Transactor,
	authenticator authn.Authenticator,
	urlProvider urlprovider.Provider,
	authorizer authz.Authorizer,
	auditService audit.Service,
	artifactStore store.ArtifactRepository,
	webhooksRepository store.WebhooksRepository,
	webhooksExecutionRepository store.WebhooksExecutionRepository,
	webhookService *registrywebhook.Service,
	spacePathStore corestore.SpacePathStore,
	artifactEventReporter *registryevents.Reporter,
	downloadStatRepository store.DownloadStatRepository,
	gitnessConfig *types.Config,
	registryBlobsDao store.RegistryBlobRepository,
	postProcessingReporter *registrypostprocessingevents.Reporter,
) harness.APIHandler {
	return harness.NewAPIHandler(
		repoDao,
		fileManager,
		upstreamproxyDao,
		tagDao,
		manifestDao,
		cleanupPolicyDao,
		imageDao,
		driver,
		config.APIURL,
		spaceFinder,
		tx,
		authenticator,
		urlProvider,
		authorizer,
		auditService,
		artifactStore,
		webhooksRepository,
		webhooksExecutionRepository,
		*webhookService,
		spacePathStore,
		*artifactEventReporter,
		downloadStatRepository,
		gitnessConfig,
		registryBlobsDao,
		postProcessingReporter,
	)
}

func OCIHandlerProvider(handlerV2 *hoci.Handler) oci.RegistryOCIHandler {
	return oci.NewOCIHandler(handlerV2)
}

func MavenHandlerProvider(handler *maven.Handler) mavenRouter.Handler {
	return mavenRouter.NewMavenHandler(handler)
}

func GenericHandlerProvider(handler *generic.Handler) generic2.Handler {
	return generic2.NewGenericArtifactHandler(handler)
}

func PackageHandlerProvider(
	handler packages.Handler,
	mavenHandler *maven.Handler,
	genericHandler *generic.Handler,
	pypiHandler python.Handler,
	nugetHandler nuget.Handler,
	npmHandler npm.Handler,
	rpmHandler rpm.Handler,
	cargoHandler cargo.Handler,
) packagerrouter.Handler {
	return packagerrouter.NewRouter(
		handler,
		mavenHandler,
		genericHandler,
		pypiHandler,
		nugetHandler,
		npmHandler,
		rpmHandler,
		cargoHandler,
	)
}

var WireSet = wire.NewSet(APIHandlerProvider, OCIHandlerProvider, AppRouterProvider,
	MavenHandlerProvider, GenericHandlerProvider, PackageHandlerProvider)
