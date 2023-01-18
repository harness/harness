// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
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
	principalStore store.PrincipalStore
	gitRPCClient   gitrpc.Interface
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
	principalStore store.PrincipalStore,
	gitRPCClient gitrpc.Interface,
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
		principalStore: principalStore,
		gitRPCClient:   gitRPCClient,
	}
}

// CreateRPCWriteParams creates base write parameters for gitrpc write operations.
// IMPORTANT: session & repo are assumed to be not nil!
func CreateRPCWriteParams(ctx context.Context, urlProvider *url.Provider,
	session *auth.Session, repo *types.Repository) (gitrpc.WriteParams, error) {
	requestID, ok := request.RequestIDFrom(ctx)
	if !ok {
		// best effort retrieving of requestID - log in case we can't find it but don't fail operation.
		log.Ctx(ctx).Warn().Msg("operation doesn't have a requestID in the context.")
	}

	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(&githook.Payload{
		APIBaseURL:  urlProvider.GetAPIBaseURLInternal(),
		RepoID:      repo.ID,
		PrincipalID: session.Principal.ID,
		RequestID:   requestID,
	})
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
