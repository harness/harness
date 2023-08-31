// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/livelog"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db             *sqlx.DB
	authorizer     authz.Authorizer
	executionStore store.ExecutionStore
	repoStore      store.RepoStore
	pipelineStore  store.PipelineStore
	stageStore     store.StageStore
	stepStore      store.StepStore
	logStore       store.LogStore
	logStream      livelog.LogStream
}

func NewController(
	db *sqlx.DB,
	authorizer authz.Authorizer,
	executionStore store.ExecutionStore,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	stageStore store.StageStore,
	stepStore store.StepStore,
	logStore store.LogStore,
	logStream livelog.LogStream,
) *Controller {
	return &Controller{
		db:             db,
		authorizer:     authorizer,
		executionStore: executionStore,
		repoStore:      repoStore,
		pipelineStore:  pipelineStore,
		stageStore:     stageStore,
		stepStore:      stepStore,
		logStore:       logStore,
		logStream:      logStream,
	}
}
