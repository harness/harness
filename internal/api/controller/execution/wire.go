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

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(db *sqlx.DB,
	authorizer authz.Authorizer,
	executionStore store.ExecutionStore,
	canceler canceler.Canceler,
	commitService commit.CommitService,
	triggerer triggerer.Triggerer,
	repoStore store.RepoStore,
	stageStore store.StageStore,
	pipelineStore store.PipelineStore,
) *Controller {
	return NewController(db, authorizer, executionStore, canceler, commitService,
		triggerer, repoStore, stageStore, pipelineStore)
}
