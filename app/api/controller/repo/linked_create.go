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
	"strings"
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

	ConnectorRef string `json:"connector_ref"`
	// RepoIdentifier is the provider-side full repo path (e.g. "owner/repo",
	// "group/subgroup/project"). Required for account-level connectors, must
	// be empty for repo-level ones; validated in the connector service.
	RepoIdentifier string `json:"repo_identifier"`
}

func (in *LinkedCreateInput) sanitize() error {
	if err := ValidateParentRef(in.ParentRef); err != nil {
		return err
	}

	if in.ConnectorRef == "" {
		return errors.InvalidArgument("connector_ref must not be empty")
	}

	in.RepoIdentifier = strings.TrimSpace(in.RepoIdentifier)

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

	connectorPath, connectorIdentifier := c.connectorService.ResolveConnectorRef(parentSpace.Path, in.ConnectorRef)
	connector := importer.ConnectorDef{
		Path:           connectorPath,
		Identifier:     connectorIdentifier,
		RepoIdentifier: in.RepoIdentifier,
	}

	access, err := c.verifyConnectorAccess(ctx, connector)
	if err != nil {
		return nil, errors.InvalidArgument("Failed to use connector to access the remote repository.")
	}

	// Register the provider-side webhook before creating any gitness state.
	// Failing here means no repo/linked_repo rows exist, so there is nothing
	// to roll back. The upsert is URL-idempotent on the SCM-service side, so
	// retrying with the same connector + URL returns the same hook.
	if err := c.webhookService.UpsertWebhook(ctx, importer.UpsertWebhookInput{
		SpacePath:           parentSpace.Path,
		ConnectorPath:       connector.Path,
		ConnectorIdentifier: connector.Identifier,
		CloneURL:            access.CloneURL,
	}); err != nil {
		return nil, webhookRegistrationUserError(err)
	}

	repo := importer.NewRepo(
		parentSpace.ID,
		parentSpace.Path,
		in.Identifier,
		in.Description,
		&session.Principal,
		access.DefaultBranch,
	)
	repo.RootSpaceID = parentSpace.RootSpaceID

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

		linkedRepo := &types.LinkedRepo{
			RepoID:              repo.ID,
			Version:             0,
			Created:             now,
			Updated:             now,
			LastFullSync:        now,
			ConnectorPath:       connector.Path,
			ConnectorIdentifier: connector.Identifier,
			ConnectorRepo:       connector.RepoIdentifier,
			ProviderRepoID:      access.ProviderRepoID,
			ProviderType:        string(access.ProviderType),
		}
		err = c.linkedRepoStore.Create(ctx, linkedRepo)
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

type connectorAccess struct {
	DefaultBranch  string
	CloneURL       string
	ProviderRepoID string
	ProviderType   importer.ProviderType
}

// verifyConnectorAccess verifies the connector can reach the remote repo and
// returns the resolved details needed by the linked-create flow.
func (c *Controller) verifyConnectorAccess(
	ctx context.Context,
	connector importer.ConnectorDef,
) (connectorAccess, error) {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal
	gitIdentity := identityFromPrincipal(systemPrincipal)

	accessInfo, err := c.connectorService.GetAccessInfo(ctx, connector)
	if err != nil {
		return connectorAccess{}, fmt.Errorf("failed to get repository access info from connector: %w", err)
	}

	urlWithCredentials, err := accessInfo.URLWithCredentials()
	if err != nil {
		return connectorAccess{}, fmt.Errorf("failed to get repository URL: %w", err)
	}

	envVars, err := githook.GenerateEnvironmentVariablesForOperation(
		ctx,
		c.urlProvider.GetInternalAPIURL(ctx),
		0,
		systemPrincipal.ID,
		true,
		enum.GitOpTypeManageRepo,
	)
	if err != nil {
		return connectorAccess{}, fmt.Errorf("failed to generate git hook environment variables: %w", err)
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
		return connectorAccess{}, fmt.Errorf("failed to create repository: %w", err)
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
		return connectorAccess{}, fmt.Errorf("failed to get repository default branch: %w", err)
	}

	providerInfo, err := c.connectorService.FetchProviderRepoInfo(ctx, connector)
	if err != nil {
		return connectorAccess{}, fmt.Errorf("failed to fetch provider repo info: %w", err)
	}

	return connectorAccess{
		DefaultBranch:  respDefBranch.BranchName,
		CloneURL:       accessInfo.URL,
		ProviderRepoID: providerInfo.RepoID,
		ProviderType:   providerInfo.Type,
	}, nil
}
