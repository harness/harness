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

package harness

import (
	"net/http"

	middlewareauthn "github.com/harness/gitness/app/api/middleware/authn"
	"github.com/harness/gitness/app/api/middleware/encode"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/middleware"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	registryevents "github.com/harness/gitness/registry/app/events"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	registrywebhook "github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/go-chi/chi/v5"
)

var (
	// terminatedPathPrefixesAPI is the list of prefixes that will require resolving terminated paths.
	terminatedPathPrefixesAPI = []string{
		"/api/v1/spaces/", "/api/v1/registry/",
	}

	// terminatedPathRegexPrefixesAPI is the list of regex prefixes that will require resolving terminated paths.
	terminatedPathRegexPrefixesAPI = []string{
		"^/api/v1/registry/([^/]+)/artifact/",
	}
)

type APIHandler interface {
	http.Handler
}

func NewAPIHandler(
	repoDao store.RegistryRepository,
	fileManager filemanager.FileManager,
	upstreamproxyDao store.UpstreamProxyConfigRepository,
	tagDao store.TagRepository,
	manifestDao store.ManifestRepository,
	cleanupPolicyDao store.CleanupPolicyRepository,
	imageDao store.ImageRepository,
	driver storagedriver.StorageDriver,
	baseURL string,
	spaceFinder refcache.SpaceFinder,
	tx dbtx.Transactor,
	authenticator authn.Authenticator,
	urlProvider urlprovider.Provider,
	authorizer authz.Authorizer,
	auditService audit.Service,
	artifactStore store.ArtifactRepository,
	webhooksRepository store.WebhooksRepository,
	webhooksExecutionRepository store.WebhooksExecutionRepository,
	webhookService registrywebhook.Service,
	spacePathStore corestore.SpacePathStore,
	artifactEventReporter registryevents.Reporter,
	downloadStatRepository store.DownloadStatRepository,
) APIHandler {
	r := chi.NewRouter()
	r.Use(audit.Middleware())
	r.Use(middlewareauthn.Attempt(authenticator))
	r.Use(middleware.CheckAuth())
	registryMetadataHelper := metadata.NewRegistryMetadataHelper(spacePathStore, spaceFinder, repoDao)
	apiController := metadata.NewAPIController(
		repoDao,
		fileManager,
		nil,
		nil,
		upstreamproxyDao,
		tagDao,
		manifestDao,
		cleanupPolicyDao,
		imageDao,
		driver,
		spaceFinder,
		tx,
		urlProvider,
		authorizer,
		auditService,
		artifactStore,
		webhooksRepository,
		webhooksExecutionRepository,
		registryMetadataHelper,
		&webhookService,
		artifactEventReporter,
		downloadStatRepository,
	)

	handler := artifact.NewStrictHandler(apiController, []artifact.StrictMiddlewareFunc{})
	muxHandler := artifact.HandlerFromMuxWithBaseURL(handler, r, baseURL)
	return encode.TerminatedPathBefore(
		terminatedPathPrefixesAPI,
		encode.TerminatedRegexPathBefore(terminatedPathRegexPrefixesAPI, muxHandler),
	)
}
