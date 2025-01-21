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
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/handler/maven"
	hoci "github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/router/harness"
	mavenRouter "github.com/harness/gitness/registry/app/api/router/maven"
	"github.com/harness/gitness/registry/app/api/router/oci"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

func AppRouterProvider(
	ocir oci.RegistryOCIHandler,
	appHandler harness.APIHandler,
	mavenHandler mavenRouter.Handler,
) AppRouter {
	return GetAppRouter(ocir, appHandler, config.APIURL, mavenHandler)
}

func APIHandlerProvider(
	repoDao store.RegistryRepository,
	upstreamproxyDao store.UpstreamProxyConfigRepository,
	tagDao store.TagRepository,
	manifestDao store.ManifestRepository,
	cleanupPolicyDao store.CleanupPolicyRepository,
	imageDao store.ImageRepository,
	driver storagedriver.StorageDriver,
	spaceStore corestore.SpaceStore,
	tx dbtx.Transactor,
	authenticator authn.Authenticator,
	urlProvider urlprovider.Provider,
	authorizer authz.Authorizer,
	auditService audit.Service,
	spacePathStore corestore.SpacePathStore,
	artifactStore store.ArtifactRepository,
) harness.APIHandler {
	return harness.NewAPIHandler(
		repoDao,
		upstreamproxyDao,
		tagDao,
		manifestDao,
		cleanupPolicyDao,
		imageDao,
		driver,
		config.APIURL,
		spaceStore,
		tx,
		authenticator,
		urlProvider,
		authorizer,
		auditService,
		spacePathStore,
		artifactStore,
	)
}

func OCIHandlerProvider(handlerV2 *hoci.Handler) oci.RegistryOCIHandler {
	return oci.NewOCIHandler(handlerV2)
}

func MavenHandlerProvider(handler *maven.Handler) mavenRouter.Handler {
	return mavenRouter.NewMavenHandler(handler)
}

var WireSet = wire.NewSet(APIHandlerProvider, OCIHandlerProvider, AppRouterProvider, MavenHandlerProvider)
