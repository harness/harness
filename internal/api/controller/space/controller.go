// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
		pipelineStore:   pipelineStore,
		secretStore:     secretStore,
		connectorStore:  connectorStore,
		templateStore:   templateStore,
		spaceStore:      spaceStore,
		repoStore:       repoStore,
		principalStore:  principalStore,
		repoCtrl:        repoCtrl,
		membershipStore: membershipStore,
		importer:        importer,
		exporter:        exporter,
	}
}
