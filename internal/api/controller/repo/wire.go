// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/services/importer"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(config *types.Config, db *sqlx.DB, urlProvider *url.Provider,
	uidCheck check.PathUID, authorizer authz.Authorizer, repoStore store.RepoStore,
	spaceStore store.SpaceStore, pipelineStore store.PipelineStore,
	principalStore store.PrincipalStore, rpcClient gitrpc.Interface,
	importer *importer.Repository,
) *Controller {
	return NewController(config.Git.DefaultBranch, db, urlProvider,
		uidCheck, authorizer, repoStore,
		spaceStore, pipelineStore, principalStore, rpcClient,
		importer)
}
