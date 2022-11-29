// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db           *sqlx.DB
	authorizer   authz.Authorizer
	pullreqStore store.PullReqStore
	repoStore    store.RepoStore
	saStore      store.ServiceAccountStore
	gitRPCClient gitrpc.Interface
}

func NewController(
	db *sqlx.DB,
	authorizer authz.Authorizer,
	pullreqStore store.PullReqStore,
	repoStore store.RepoStore,
	saStore store.ServiceAccountStore,
	gitRPCClient gitrpc.Interface,
) *Controller {
	return &Controller{
		db:           db,
		authorizer:   authorizer,
		pullreqStore: pullreqStore,
		repoStore:    repoStore,
		saStore:      saStore,
		gitRPCClient: gitRPCClient,
	}
}
