// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repo

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	defaultBranch     string
	tx                dbtx.Transactor
	urlProvider       url.Provider
	uidCheck          check.PathUID
	authorizer        authz.Authorizer
	repoStore         store.RepoStore
	spaceStore        store.SpaceStore
	pipelineStore     store.PipelineStore
	principalStore    store.PrincipalStore
	ruleStore         store.RuleStore
	protectionManager *protection.Manager
	gitRPCClient      gitrpc.Interface
	importer          *importer.Repository
	codeOwners        *codeowners.Service
}

func NewController(
	defaultBranch string,
	tx dbtx.Transactor,
	urlProvider url.Provider,
	uidCheck check.PathUID,
	authorizer authz.Authorizer,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
	pipelineStore store.PipelineStore,
	principalStore store.PrincipalStore,
	ruleStore store.RuleStore,
	protectionManager *protection.Manager,
	gitRPCClient gitrpc.Interface,
	importer *importer.Repository,
	codeOwners *codeowners.Service,
) *Controller {
	return &Controller{
		defaultBranch:     defaultBranch,
		tx:                tx,
		urlProvider:       urlProvider,
		uidCheck:          uidCheck,
		authorizer:        authorizer,
		repoStore:         repoStore,
		spaceStore:        spaceStore,
		pipelineStore:     pipelineStore,
		principalStore:    principalStore,
		ruleStore:         ruleStore,
		protectionManager: protectionManager,
		gitRPCClient:      gitRPCClient,
		importer:          importer,
		codeOwners:        codeOwners,
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
func CreateRPCWriteParams(ctx context.Context, urlProvider url.Provider,
	session *auth.Session, repo *types.Repository) (gitrpc.WriteParams, error) {
	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		urlProvider.GetInternalAPIURL(),
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
