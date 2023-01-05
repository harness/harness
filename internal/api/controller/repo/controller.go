// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
)

type Controller struct {
	defaultBranch  string
	gitBaseURL     string
	repoCheck      check.Repo
	authorizer     authz.Authorizer
	spaceStore     store.SpaceStore
	repoStore      store.RepoStore
	principalStore store.PrincipalStore
	gitRPCClient   gitrpc.Interface
}

func NewController(
	defaultBranch string,
	gitBaseURL string,
	repoCheck check.Repo,
	authorizer authz.Authorizer,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	principalStore store.PrincipalStore,
	gitRPCClient gitrpc.Interface,
) *Controller {
	return &Controller{
		defaultBranch:  defaultBranch,
		gitBaseURL:     gitBaseURL,
		repoCheck:      repoCheck,
		authorizer:     authorizer,
		spaceStore:     spaceStore,
		repoStore:      repoStore,
		principalStore: principalStore,
		gitRPCClient:   gitRPCClient,
	}
}

// CreateRPCWriteParams creates base write parameters for gitrpc write operations.
// IMPORTANT: session & repo are assumed to be not nil!
func CreateRPCWriteParams(session *auth.Session, repo *types.Repository) gitrpc.WriteParams {
	// generate envars (add everything githook CLI needs for execution)
	// TODO: envVars := githook.GenerateGitHookEnvironmentVariables(repo, session.Principal)
	envVars := map[string]string{}

	return gitrpc.WriteParams{
		Actor: gitrpc.Identity{
			Name:  session.Principal.DisplayName,
			Email: session.Principal.Email,
		},
		RepoUID: repo.GitUID,
		EnvVars: envVars,
	}
}

// CreateRPCReadParams creates base read parameters for gitrpc read operations.
// IMPORTANT: repo is assumed to be not nil!
func CreateRPCReadParams(repo *types.Repository) gitrpc.ReadParams {
	return gitrpc.ReadParams{
		RepoUID: repo.GitUID,
	}
}
