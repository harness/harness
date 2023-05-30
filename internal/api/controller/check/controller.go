// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	db            *sqlx.DB
	authorizer    authz.Authorizer
	checkStore    store.CheckStore
	reqCheckStore store.ReqCheckStore
	repoStore     store.RepoStore
	gitRPCClient  gitrpc.Interface
}

func NewController(
	db *sqlx.DB,
	authorizer authz.Authorizer,
	checkStore store.CheckStore,
	repoStore store.RepoStore,
	gitRPCClient gitrpc.Interface,
) *Controller {
	return &Controller{
		db:           db,
		authorizer:   authorizer,
		checkStore:   checkStore,
		repoStore:    repoStore,
		gitRPCClient: gitRPCClient,
	}
}

func (c *Controller) getRepoCheckAccess(ctx context.Context,
	session *auth.Session, repoRef string, reqPermission enum.Permission,
) (*types.Repository, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission, false); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return repo, nil
}
