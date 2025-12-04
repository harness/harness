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
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
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

	connector := in.Connector

	// The importer job requires provider for execution.
	provider, err := c.connectorService.AsProvider(ctx, connector)
	if err != nil {
		return nil, fmt.Errorf("failed to convert connector to provider: %w", err)
	}

	// From connector info we need to get remote repository info and importer.Provider.
	// Repository info we need to create repository in the DB.
	remoteRepository, provider, err := importer.LoadRepositoryFromProvider(ctx, provider, connector.Repo)
	if err != nil {
		return nil, errors.InvalidArgument("Failed to get access to the remote repository.")
	}

	repo, isPublic := remoteRepository.ToRepo(
		parentSpace.ID,
		parentSpace.Path,
		in.Identifier,
		in.Description,
		&session.Principal,
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
			ConnectorRepo:       in.Connector.Repo,
		})
		if err != nil {
			return fmt.Errorf("failed to create linked repository: %w", err)
		}

		err = c.importLinked.Run(ctx,
			provider,
			repo,
			isPublic,
			remoteRepository.CloneURL,
		)
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
