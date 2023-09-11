// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"github.com/harness/gitness/internal/services/exporter"
	"strconv"
	"strings"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/internal/services/importer"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

type Controller struct {
	defaultBranch  string
	db             *sqlx.DB
	urlProvider    *url.Provider
	uidCheck       check.PathUID
	authorizer     authz.Authorizer
	pathStore      store.PathStore
	repoStore      store.RepoStore
	spaceStore     store.SpaceStore
	pipelineStore  store.PipelineStore
	principalStore store.PrincipalStore
	gitRPCClient   gitrpc.Interface
	importer       *importer.Repository
	exporter       *exporter.Repository
}

func NewController(
	defaultBranch string,
	db *sqlx.DB,
	urlProvider *url.Provider,
	uidCheck check.PathUID,
	authorizer authz.Authorizer,
	pathStore store.PathStore,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
	pipelineStore store.PipelineStore,
	principalStore store.PrincipalStore,
	gitRPCClient gitrpc.Interface,
	importer *importer.Repository,
) *Controller {
	return &Controller{
		defaultBranch:  defaultBranch,
		db:             db,
		urlProvider:    urlProvider,
		uidCheck:       uidCheck,
		authorizer:     authorizer,
		pathStore:      pathStore,
		repoStore:      repoStore,
		spaceStore:     spaceStore,
		pipelineStore:  pipelineStore,
		principalStore: principalStore,
		gitRPCClient:   gitRPCClient,
		importer:       importer,
	}
}

// getRepoCheckAccess fetches an active repo (not one that is currently being imported)
// and checks if the current user has permission to access it.
func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
	orPublic bool,
) (*types.Repository, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if repo.Importing {
		return nil, usererror.BadRequest("Repository import is in progress.")
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission, orPublic); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return repo, nil
}

// CreateRPCWriteParams creates base write parameters for gitrpc write operations.
// IMPORTANT: session & repo are assumed to be not nil!
func CreateRPCWriteParams(ctx context.Context, urlProvider *url.Provider,
	session *auth.Session, repo *types.Repository) (gitrpc.WriteParams, error) {
	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		urlProvider.GetAPIBaseURLInternal(),
		repo.ID,
		session.Principal.ID,
		false,
	)
	if err != nil {
		return gitrpc.WriteParams{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	return gitrpc.WriteParams{
		Actor: gitrpc.Identity{
			Name:  session.Principal.DisplayName,
			Email: session.Principal.Email,
		},
		RepoUID: repo.GitUID,
		EnvVars: envVars,
	}, nil
}

// CreateRPCReadParams creates base read parameters for gitrpc read operations.
// IMPORTANT: repo is assumed to be not nil!
func CreateRPCReadParams(repo *types.Repository) gitrpc.ReadParams {
	return gitrpc.ReadParams{
		RepoUID: repo.GitUID,
	}
}

func (c *Controller) validateParentRef(parentRef string) error {
	parentRefAsID, err := strconv.ParseInt(parentRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(parentRef)) == 0) {
		return errRepositoryRequiresParent
	}

	return nil
}
