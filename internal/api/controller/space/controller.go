// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types/check"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db             *sqlx.DB
	urlProvider    *url.Provider
	uidCheck       check.PathUID
	authorizer     authz.Authorizer
	pathStore      store.PathStore
	spaceStore     store.SpaceStore
	repoStore      store.RepoStore
	principalStore store.PrincipalStore
	repoCtrl       *repo.Controller
}

func NewController(db *sqlx.DB, urlProvider *url.Provider,
	uidCheck check.PathUID, authorizer authz.Authorizer,
	pathStore store.PathStore, spaceStore store.SpaceStore,
	repoStore store.RepoStore, principalStore store.PrincipalStore, repoCtrl *repo.Controller,
) *Controller {
	return &Controller{
		db:             db,
		urlProvider:    urlProvider,
		uidCheck:       uidCheck,
		authorizer:     authorizer,
		pathStore:      pathStore,
		spaceStore:     spaceStore,
		repoStore:      repoStore,
		principalStore: principalStore,
		repoCtrl:       repoCtrl,
	}
}
