// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/githook"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/services/importer"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type ImportInput struct {
	ParentRef string `json:"parent_ref"`
	UID       string `json:"uid"`

	Provider    importer.ProviderType `json:"provider"`
	ProviderURL string                `json:"provider_url"`
	RepoSlug    string                `json:"repo_slug"`
	Username    string                `json:"username"`
	Password    string                `json:"password"`

	Description string `json:"description"`
}

// Import creates a new empty repository and starts git import to it from a remote repository.
func (c *Controller) Import(ctx context.Context, session *auth.Session, in *ImportInput) (*types.Repository, error) {
	parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	err = c.sanitizeImportInput(in)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	providerInfo := importer.ProviderInfo{
		Type: in.Provider,
		Host: in.ProviderURL,
		User: in.Username,
		Pass: in.Password,
	}

	repoInfo, err := importer.Repo(ctx, providerInfo, in.RepoSlug)
	if err != nil {
		return nil, err
	}

	jobUID, err := job.UID()
	if err != nil {
		return nil, fmt.Errorf("error creating job UID: %w", err)
	}

	gitRPCResp, err := c.createEmptyGitRepository(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("error creating repository on GitRPC: %w", err)
	}

	now := time.Now().UnixMilli()
	repo := &types.Repository{
		Version:         0,
		ParentID:        parentSpace.ID,
		UID:             in.UID,
		GitUID:          gitRPCResp.UID,
		Path:            "", // the path is set in the DB transaction below
		Description:     in.Description,
		IsPublic:        repoInfo.IsPublic,
		CreatedBy:       session.Principal.ID,
		Created:         now,
		Updated:         now,
		ForkID:          0,
		DefaultBranch:   repoInfo.DefaultBranch,
		Importing:       true,
		ImportingJobUID: &jobUID,
	}

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		// lock parent space path to ensure it doesn't get updated while we setup new repo
		spacePath, err := c.pathStore.FindPrimaryWithLock(ctx, enum.PathTargetTypeSpace, parentSpace.ID)
		if err != nil {
			return usererror.BadRequest("Parent not found'")
		}

		repo.Path = paths.Concatinate(spacePath.Value, in.UID)

		err = c.repoStore.Create(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to create repository in storage: %w", err)
		}

		path := &types.Path{
			Version:    0,
			Value:      repo.Path,
			IsPrimary:  true,
			TargetType: enum.PathTargetTypeRepo,
			TargetID:   repo.ID,
			CreatedBy:  repo.CreatedBy,
			Created:    repo.Created,
			Updated:    repo.Updated,
		}

		err = c.pathStore.Create(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}

		return nil
	})
	if err != nil {
		if err := c.DeleteRepositoryRPC(ctx, session, repo); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("gitrpc failed to delete repo for cleanup")
		}

		return nil, err
	}

	err = c.importer.Run(ctx, jobUID, importer.Input{
		RepoID:   repo.ID,
		GitUser:  in.Username,
		GitPass:  in.Password,
		CloneURL: repoInfo.CloneURL,
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to start import repository job")
	}

	repo.GitURL = c.urlProvider.GenerateRepoCloneURL(repo.Path)

	return repo, nil
}

func (c *Controller) sanitizeImportInput(in *ImportInput) error {
	if err := c.validateParentRef(in.ParentRef); err != nil {
		return err
	}

	if err := c.uidCheck(in.UID, false); err != nil {
		return err
	}

	if in.Provider == "" {
		return usererror.BadRequest("provider must be provided")
	}

	if in.RepoSlug == "" {
		return usererror.BadRequest("repo slug must be provided")
	}

	return nil
}

func (c *Controller) createEmptyGitRepository(
	ctx context.Context,
	session *auth.Session,
) (*gitrpc.CreateRepositoryOutput, error) {
	// generate envars (add everything githook CLI needs for execution)
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		c.urlProvider.GetAPIBaseURLInternal(),
		0,
		session.Principal.ID,
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	actor := rpcIdentityFromPrincipal(session.Principal)
	committer := rpcIdentityFromPrincipal(bootstrap.NewSystemServiceSession().Principal)
	now := time.Now()

	resp, err := c.gitRPCClient.CreateRepository(ctx, &gitrpc.CreateRepositoryParams{
		Actor:         *actor,
		EnvVars:       envVars,
		DefaultBranch: c.defaultBranch,
		Files:         nil,
		Author:        actor,
		AuthorDate:    &now,
		Committer:     committer,
		CommitterDate: &now,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create repo on gitrpc: %w", err)
	}

	return resp, nil
}
