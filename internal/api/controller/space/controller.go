// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package space

import (
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/services/exporter"
	"github.com/harness/gitness/internal/services/importer"
	"github.com/harness/gitness/internal/sse"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/jmoiron/sqlx"
)

var (
	// TODO (Nested Spaces): Remove once full support is added
	errNestedSpacesNotSupported = usererror.BadRequestf("Nested spaces are not supported.")
)

type Controller struct {
	nestedSpacesEnabled bool

	db              *sqlx.DB
	urlProvider     *url.Provider
	sseStreamer     sse.Streamer
	uidCheck        check.PathUID
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
}

func NewController(config *types.Config, db *sqlx.DB, urlProvider *url.Provider,
	sseStreamer sse.Streamer, uidCheck check.PathUID, authorizer authz.Authorizer,
	spacePathStore store.SpacePathStore, pipelineStore store.PipelineStore, secretStore store.SecretStore,
	connectorStore store.ConnectorStore, templateStore store.TemplateStore, spaceStore store.SpaceStore,
	repoStore store.RepoStore, principalStore store.PrincipalStore, repoCtrl *repo.Controller,
	membershipStore store.MembershipStore, importer *importer.Repository, exporter *exporter.Repository,
) *Controller {
	return &Controller{
		nestedSpacesEnabled: config.NestedSpacesEnabled,
		db:                  db,
		urlProvider:         urlProvider,
		sseStreamer:         sseStreamer,
		uidCheck:            uidCheck,
		authorizer:          authorizer,
		spacePathStore:      spacePathStore,
		pipelineStore:       pipelineStore,
		secretStore:         secretStore,
		connectorStore:      connectorStore,
		templateStore:       templateStore,
		spaceStore:          spaceStore,
		repoStore:           repoStore,
		principalStore:      principalStore,
		repoCtrl:            repoCtrl,
		membershipStore:     membershipStore,
		importer:            importer,
		exporter:            exporter,
	}
}
