// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/pipeline/events"
	"github.com/harness/gitness/internal/services/exporter"
	"github.com/harness/gitness/internal/services/importer"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types/check"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db              *sqlx.DB
	urlProvider     *url.Provider
	eventsStream    events.EventsStreamer
	uidCheck        check.PathUID
	authorizer      authz.Authorizer
	pathStore       store.PathStore
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

func NewController(db *sqlx.DB, urlProvider *url.Provider, eventsStream events.EventsStreamer,
	uidCheck check.PathUID, authorizer authz.Authorizer,
	pathStore store.PathStore, pipelineStore store.PipelineStore, secretStore store.SecretStore,
	connectorStore store.ConnectorStore, templateStore store.TemplateStore, spaceStore store.SpaceStore,
	repoStore store.RepoStore, principalStore store.PrincipalStore, repoCtrl *repo.Controller,
	membershipStore store.MembershipStore, importer *importer.Repository,
) *Controller {
	return &Controller{
		db:              db,
		urlProvider:     urlProvider,
		eventsStream:    eventsStream,
		uidCheck:        uidCheck,
		authorizer:      authorizer,
		pathStore:       pathStore,
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
	}
}
