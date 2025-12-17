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
	"time"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type LinkedCreateInput struct {
	ParentRef   string `json:"parent_ref"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`

	IsPublic bool `json:"is_public"`

	Connector importer.ConnectorDef `json:"connector"`
}

func (in *LinkedCreateInput) sanitize() error {
	if err := ValidateParentRef(in.ParentRef); err != nil {
		return err
	}

	return nil
}

func (c *Controller) LinkedCreate(
	ctx context.Context,
	session *auth.Session,
	in *LinkedCreateInput,
) (*RepositoryOutput, error) {
	if err := in.sanitize(); err != nil {
		return nil, err
	}

	if err := c.identifierCheck(in.Identifier, session); err != nil {
		return nil, err
	}

	parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	isPublicAccessSupported, err := c.publicAccess.
		IsPublicAccessSupported(ctx, enum.PublicResourceTypeRepo, parentSpace.Path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to check if public access is supported for parent space %q: %w",
			parentSpace.Path,
			err,
		)
	}
	if in.IsPublic && !isPublicAccessSupported {
		return nil, errPublicRepoCreationDisabled
	}

	defaultBranch, err := c.verifyConnectorAccess(ctx, in.Connector)
	if err != nil {
		return nil, errors.InvalidArgument("Failed to use connector to access the remote repository.")
	}

	repo := importer.NewRepo(
		parentSpace.ID,
		parentSpace.Path,
		in.Identifier,
		in.Description,
		&session.Principal,
		defaultBranch,
	)

	repo.Type = enum.RepoTypeLinked

	now := time.Now().UnixMilli()

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.resourceLimiter.RepoCount(ctx, parentSpace.ID, 1); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		// lock the space for update during repo creation to prevent racing conditions with space soft delete.
		_, err = c.spaceStore.FindForUpdate(ctx, parentSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to find the parent space: %w", err)
		}

		err = c.repoStore.Create(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}

		err = c.linkedRepoStore.Create(ctx, &types.LinkedRepo{
			RepoID:              repo.ID,
			Version:             0,
			Created:             now,
			Updated:             now,
			LastFullSync:        now,
			ConnectorPath:       in.Connector.Path,
			ConnectorIdentifier: in.Connector.Identifier,
		})
		if err != nil {
			return fmt.Errorf("failed to create linked repository: %w", err)
		}

		err = c.importLinked.Run(ctx, repo.ID, in.IsPublic)
		if err != nil {
			return fmt.Errorf("failed to start link repository job: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
	repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRepository, repo.Identifier),
		audit.ActionCreated,
		paths.Parent(repo.Path),
		audit.WithNewObject(audit.RepositoryObject{
			Repository: *repo,
			IsPublic:   false,
		}),
	)
	if err != nil {
		log.Warn().Msgf("failed to insert audit log for import repository operation: %s", err)
	}

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeRepositoryCreate,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      repo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:           repo.ID,
			instrument.PropertyRepositoryName:         repo.Identifier,
			instrument.PropertyRepositoryCreationType: instrument.CreationTypeLink,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for import repository operation: %s", err)
	}

	repoOutput, err := GetRepoOutputWithAccess(ctx, c.repoFinder, false, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo output: %w", err)
	}

	return repoOutput, nil
}

func (c *Controller) verifyConnectorAccess(ctx context.Context, connector importer.ConnectorDef) (string, error) {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal
	gitIdentity := identityFromPrincipal(systemPrincipal)

	accessInfo, err := c.connectorService.GetAccessInfo(ctx, connector)
	if err != nil {
		return "", fmt.Errorf("failed to get repository access info from connector: %w", err)
	}

	urlWithCredentials, err := accessInfo.URLWithCredentials()
	if err != nil {
		return "", fmt.Errorf("failed to get repository URL: %w", err)
	}

	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		c.urlProvider.GetInternalAPIURL(ctx),
		0,
		systemPrincipal.ID,
		true,
		true,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}

	now := time.Now()
	resp, err := c.git.CreateRepository(ctx, &git.CreateRepositoryParams{
		Actor:         *gitIdentity,
		EnvVars:       envVars,
		DefaultBranch: "main",
		Files:         nil,
		Author:        gitIdentity,
		AuthorDate:    &now,
		Committer:     gitIdentity,
		CommitterDate: &now,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create repository: %w", err)
	}

	gitUID := resp.UID

	writeParams := git.WriteParams{
		Actor:   *gitIdentity,
		RepoUID: gitUID,
		EnvVars: envVars,
	}

	defer func() {
		if errDel := c.git.DeleteRepository(context.WithoutCancel(ctx), &git.DeleteRepositoryParams{
			WriteParams: writeParams,
		}); errDel != nil {
			log.Warn().Err(errDel).
				Msg("failed to delete temporary git repository")
		}
	}()

	respDefBranch, err := c.git.GetRemoteDefaultBranch(ctx, &git.GetRemoteDefaultBranchParams{
		ReadParams: git.ReadParams{RepoUID: gitUID},
		Source:     urlWithCredentials,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get repository default branch: %w", err)
	}

	return respDefBranch.BranchName, nil
}
