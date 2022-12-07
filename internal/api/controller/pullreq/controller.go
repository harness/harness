// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"errors"
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

func (c *Controller) verifyBranchExistence(ctx context.Context, repo *types.Repository, branch string) error {
	_, err := c.gitRPCClient.GetRef(ctx,
		&gitrpc.GetRefParams{RepoUID: repo.GitUID, Name: branch, Type: gitrpc.RefTypeBranch})
	if errors.Is(err, gitrpc.ErrNotFound) {
		return usererror.BadRequest(
			fmt.Sprintf("branch %s does not exist in the repository %s", branch, repo.UID))
	}
	if err != nil {
		return fmt.Errorf(
			"failed to check existence of the branch %s in the repository %s: %w",
			branch, repo.UID, err)
	}

	return nil
}

func (c *Controller) getRepoCheckAccess(ctx context.Context,
	session *auth.Session, repoRef string, reqPermission enum.Permission) (*types.Repository, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission, false); err != nil {
		return nil, err
	}

	return repo, nil
}
