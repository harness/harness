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

package space

import (
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/exporter"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
)

var (
	// TODO (Nested Spaces): Remove once full support is added
	errNestedSpacesNotSupported    = usererror.BadRequestf("Nested spaces are not supported.")
	errPublicSpaceCreationDisabled = usererror.BadRequestf("Public space creation is disabled.")
)

type Controller struct {
	nestedSpacesEnabled           bool
	publicResourceCreationEnabled bool

	tx              dbtx.Transactor
	urlProvider     url.Provider
	sseStreamer     sse.Streamer
	identifierCheck check.SpaceIdentifier
	authorizer      authz.Authorizer
	spacePathStore  store.SpacePathStore
	pipelineStore   store.PipelineStore
	secretStore     store.SecretStore
	connectorStore  store.ConnectorStore
	templateStore   store.TemplateStore
	spaceStore      store.SpaceStore
	repoStore       store.RepoStore
	principalStore  store.PrincipalStore
	repoCtrl        *repo.Controller
	membershipStore store.MembershipStore
	importer        *importer.Repository
	exporter        *exporter.Repository
	resourceLimiter limiter.ResourceLimiter
}

func NewController(config *types.Config, tx dbtx.Transactor, urlProvider url.Provider,
	sseStreamer sse.Streamer, identifierCheck check.SpaceIdentifier, authorizer authz.Authorizer,
	spacePathStore store.SpacePathStore, pipelineStore store.PipelineStore, secretStore store.SecretStore,
	connectorStore store.ConnectorStore, templateStore store.TemplateStore, spaceStore store.SpaceStore,
	repoStore store.RepoStore, principalStore store.PrincipalStore, repoCtrl *repo.Controller,
	membershipStore store.MembershipStore, importer *importer.Repository, exporter *exporter.Repository,
	limiter limiter.ResourceLimiter,
) *Controller {
	return &Controller{
		nestedSpacesEnabled:           config.NestedSpacesEnabled,
		publicResourceCreationEnabled: config.PublicResourceCreationEnabled,
		tx:                            tx,
		urlProvider:                   urlProvider,
		sseStreamer:                   sseStreamer,
		identifierCheck:               identifierCheck,
		authorizer:                    authorizer,
		spacePathStore:                spacePathStore,
		pipelineStore:                 pipelineStore,
		secretStore:                   secretStore,
		connectorStore:                connectorStore,
		templateStore:                 templateStore,
		spaceStore:                    spaceStore,
		repoStore:                     repoStore,
		principalStore:                principalStore,
		repoCtrl:                      repoCtrl,
		membershipStore:               membershipStore,
		importer:                      importer,
		exporter:                      exporter,
		resourceLimiter:               limiter,
	}
}
