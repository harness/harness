// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/pipeline/canceler"
	"github.com/harness/gitness/internal/pipeline/commit"
	"github.com/harness/gitness/internal/pipeline/triggerer"
	"github.com/harness/gitness/internal/store"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db             *sqlx.DB
	authorizer     authz.Authorizer
	executionStore store.ExecutionStore
	checkStore     store.CheckStore
	canceler       canceler.Canceler
	commitService  commit.CommitService
	triggerer      triggerer.Triggerer
	repoStore      store.RepoStore
	stageStore     store.StageStore
	pipelineStore  store.PipelineStore
}

func NewController(
	db *sqlx.DB,
	authorizer authz.Authorizer,
	executionStore store.ExecutionStore,
	checkStore store.CheckStore,
	canceler canceler.Canceler,
	commitService commit.CommitService,
	triggerer triggerer.Triggerer,
	repoStore store.RepoStore,
	stageStore store.StageStore,
	pipelineStore store.PipelineStore,
) *Controller {
	return &Controller{
		db:             db,
		authorizer:     authorizer,
		executionStore: executionStore,
		checkStore:     checkStore,
		canceler:       canceler,
		commitService:  commitService,
		triggerer:      triggerer,
		repoStore:      repoStore,
		stageStore:     stageStore,
		pipelineStore:  pipelineStore,
	}
}
