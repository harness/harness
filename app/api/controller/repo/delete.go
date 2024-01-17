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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Delete deletes a repo.
func (c *Controller) Delete(ctx context.Context, session *auth.Session, repoRef string) error {
	// note: can't use c.getRepoCheckAccess because import job for repositories being imported must be cancelled.
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoDelete, false); err != nil {
		return err
	}

	log.Ctx(ctx).Info().
		Int64("repo.id", repo.ID).
		Str("repo.path", repo.Path).
		Msgf("deleting repository")

	if repo.Importing {
		err = c.importer.Cancel(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to cancel repository import")
		}
	}

	return c.DeleteNoAuth(ctx, session, repo)
}

func (c *Controller) DeleteNoAuth(ctx context.Context, session *auth.Session, repo *types.Repository) error {
	if err := c.deleteGitRepository(ctx, session, repo); err != nil {
		return fmt.Errorf("failed to delete git repository: %w", err)
	}

	if err := c.repoStore.Delete(ctx, repo.ID); err != nil {
		return fmt.Errorf("failed to delete repo from db: %w", err)
	}

	c.eventReporter.Deleted(
		ctx,
		&repoevents.DeletedPayload{
			RepoID: repo.ID,
		},
	)
	return nil
}

func (c *Controller) deleteGitRepository(
	ctx context.Context,
	session *auth.Session,
	repo *types.Repository,
) error {
	if repo.Importing {
		log.Ctx(ctx).Debug().Str("repo.git_uid", repo.GitUID).
			Msg("skipping removal of git directory for repository being imported")
		return nil
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return fmt.Errorf("failed to create RPC write params: %w", err)
	}

	err = c.git.DeleteRepository(ctx, &git.DeleteRepositoryParams{
		WriteParams: writeParams,
	})

	// deletion should not fail if dir does not exist in repos dir
	if errors.IsNotFound(err) {
		log.Ctx(ctx).Warn().Str("repo.git_uid", repo.GitUID).
			Msg("git repository directory does not exist")
	} else if err != nil {
		// deletion has failed before removing(rename) the repo dir
		return fmt.Errorf("failed to delete git repository directory %s: %w", repo.GitUID, err)
	}
	return nil
}
