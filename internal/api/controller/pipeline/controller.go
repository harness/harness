// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	defaultBranch string
	db            *sqlx.DB
	uidCheck      check.PathUID
	pathStore     store.PathStore
	repoStore     store.RepoStore
	authorizer    authz.Authorizer
	pipelineStore store.PipelineStore
	spaceStore    store.SpaceStore
}

func NewController(
	db *sqlx.DB,
	uidCheck check.PathUID,
	authorizer authz.Authorizer,
	pathStore store.PathStore,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	spaceStore store.SpaceStore,
) *Controller {
	return &Controller{
		db:            db,
		uidCheck:      uidCheck,
		pathStore:     pathStore,
		repoStore:     repoStore,
		authorizer:    authorizer,
		pipelineStore: pipelineStore,
		spaceStore:    spaceStore,
	}
}
