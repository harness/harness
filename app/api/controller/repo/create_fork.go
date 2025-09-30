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
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type CreateForkInput struct {
	ParentRef  string `json:"parent_ref"`
	Identifier string `json:"identifier"`
	ForkBranch string `json:"fork_branch"`
	IsPublic   *bool  `json:"is_public"`
}

//nolint:gocognit
func (c *Controller) CreateFork(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateForkInput,
) (*RepositoryOutput, error) {
	repoUpstreamCore, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	repoUpstream, err := c.repoStore.Find(ctx, repoUpstreamCore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find the upstream repo: %w", err)
	}

	if repoUpstream.IsEmpty {
		return nil, errors.InvalidArgument("Can not fork an empty repository.")
	}

	parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	isUpstreamPublic, err := c.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repoUpstream.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo public access: %w", err)
	}

	var isForkPublic bool

	if in.IsPublic == nil {
		isForkPublic = isUpstreamPublic
	} else {
		isForkPublic = *in.IsPublic
	}

	if isForkPublic {
		if !isUpstreamPublic {
			return nil, errors.InvalidArgument("Can not create a public fork from a private repository.")
		}

		isPublicAccessSupported, err := c.publicAccess.IsPublicAccessSupported(ctx, parentSpace.Path)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to check if public access is supported for parent space %q: %w",
				parentSpace.Path,
				err,
			)
		}

		if !isPublicAccessSupported {
			return nil, errPublicRepoCreationDisabled
		}
	}

	defaultBranch := repoUpstream.DefaultBranch

	if in.ForkBranch != "" && repoUpstream.DefaultBranch != in.ForkBranch {
		_, err := c.git.GetBranch(ctx, &git.GetBranchParams{
			ReadParams: git.CreateReadParams(repoUpstream),
			BranchName: in.ForkBranch,
		})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, errors.InvalidArgument("Fork branch not found.")
			}

			return nil, fmt.Errorf("failed to get branch: %w", err)
		}

		defaultBranch = in.ForkBranch
	}

	err = c.repoCheck.Create(ctx, session, &CheckInput{
		ParentRef:         parentSpace.Path,
		Identifier:        in.Identifier,
		DefaultBranch:     defaultBranch,
		Description:       repoUpstream.Description,
		IsPublic:          isForkPublic,
		IsFork:            true,
		UpstreamPath:      repoUpstream.Path,
		CreateFileOptions: CreateFileOptions{},
	})
	if err != nil {
		return nil, err
	}

	gitForkRepo, _, err := c.createGitRepository(
		ctx,
		session,
		in.Identifier,
		repoUpstream.Description,
		defaultBranch,
		CreateFileOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("error creating repository on git: %w", err)
	}

	var repoFork *types.Repository

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.resourceLimiter.RepoCount(ctx, parentSpace.ID, 1); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		// lock the space for update during repo creation to prevent racing conditions with space soft delete.
		_, err = c.spaceStore.FindForUpdate(ctx, parentSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to find the parent space: %w", err)
		}

		repoUpstream, err = c.repoStore.Find(ctx, repoUpstream.ID)
		if err != nil {
			return fmt.Errorf("failed to find the upstream repo: %w", err)
		}

		now := time.Now().UnixMilli()
		repoFork = &types.Repository{
			Version:       0,
			ParentID:      parentSpace.ID,
			Identifier:    in.Identifier,
			GitUID:        gitForkRepo.UID,
			Description:   repoUpstream.Description,
			CreatedBy:     session.Principal.ID,
			Created:       now,
			Updated:       now,
			LastGITPush:   now,
			ForkID:        repoUpstream.ID,
			DefaultBranch: defaultBranch,
			IsEmpty:       true,
			State:         enum.RepoStateGitImport,
			Tags:          json.RawMessage(`{}`),
		}

		err = c.repoStore.Create(ctx, repoFork)
		if err != nil {
			return fmt.Errorf("failed to create fork repository: %w", err)
		}

		repoUpstream.NumForks++
		err = c.repoStore.Update(ctx, repoUpstream)
		if err != nil {
			return fmt.Errorf("failed to update upstream repository: %w", err)
		}

		return nil
	}, sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		// best effort cleanup
		if dErr := c.DeleteGitRepository(ctx, session, gitForkRepo.UID); dErr != nil {
			log.Ctx(ctx).Warn().Err(dErr).Msg("failed to delete repo for cleanup")
		}
		return nil, err
	}

	// revert this when import fetches LFS objects
	if err := c.settings.RepoSet(ctx, repoFork.ID, settings.KeyGitLFSEnabled, false); err != nil {
		log.Warn().Err(err).Msg("failed to disable Git LFS in repository settings")
	}

	err = c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, repoFork.Path, isForkPublic)
	if err != nil {
		if dErr := c.publicAccess.Delete(ctx, enum.PublicResourceTypeRepo, repoFork.Path); dErr != nil {
			return nil, fmt.Errorf("failed to set repo public access (and public access cleanup: %w): %w", dErr, err)
		}

		// only cleanup repo itself if cleanup of public access succeeded (to avoid leaking public access)
		if dErr := c.PurgeNoAuth(ctx, session, repoFork); dErr != nil {
			return nil, fmt.Errorf("failed to set repo public access (and repo purge: %w): %w", dErr, err)
		}

		return nil, fmt.Errorf("failed to set repo public access (successful cleanup): %w", err)
	}

	// backfil GitURL
	repoFork.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repoFork.Path)
	repoFork.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repoFork.Path)

	repoOutput := GetRepoOutputWithAccess(ctx, isForkPublic, repoFork)

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRepository, repoFork.Identifier),
		audit.ActionCreated,
		paths.Parent(repoFork.Path),
		audit.WithNewObject(audit.RepositoryObject{
			Repository: repoOutput.Repository,
			IsPublic:   repoOutput.IsPublic,
		}),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).
			Msg("failed to insert audit log for create fork repository operation")
	}

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeRepositoryCreate,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      repoFork.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:           repoFork.ID,
			instrument.PropertyRepositoryName:         repoFork.Identifier,
			instrument.PropertyRepositoryCreationType: instrument.CreationTypeCreate,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).
			Msg("failed to insert instrumentation record for create fork repository operation")
	}

	c.eventReporter.Created(ctx, &repoevents.CreatedPayload{
		Base:     eventBase(repoFork.Core(), &session.Principal),
		IsPublic: isForkPublic,
	})

	var refSpecType importer.RefSpecType
	var ref string

	if in.ForkBranch != "" {
		refSpecType = importer.RefSpecTypeReference
		ref = "refs/heads/" + in.ForkBranch
	} else {
		refSpecType = importer.RefSpecTypeBranchesAndTags
	}

	err = c.referenceSync.Run(ctx, repoUpstream.ID, repoFork.ID, refSpecType, ref, ref)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to start job for repository reference sync")
	}

	return repoOutput, nil
}
