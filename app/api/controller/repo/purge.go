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
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/githook"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Purge removes a repo permanently.
func (c *Controller) Purge(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	deletedAt int64,
) error {
	repo, err := c.repoStore.FindByRefAndDeletedAt(ctx, repoRef, deletedAt)
	if err != nil {
		return fmt.Errorf("failed to find the repo (deleted at %d): %w", deletedAt, err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoDelete); err != nil {
		return err
	}

	log.Ctx(ctx).Info().
		Int64("repo.id", repo.ID).
		Str("repo.path", repo.Path).
		Msg("purging repository")

	if repo.Deleted == nil {
		return usererror.BadRequest("Repository has to be deleted before it can be purged.")
	}

	return c.PurgeNoAuth(ctx, session, repo)
}

func (c *Controller) PurgeNoAuth(
	ctx context.Context,
	session *auth.Session,
	repo *types.Repository,
) error {
	if repo.State == enum.RepoStateGitImport {
		log.Ctx(ctx).Info().Msg("repository is importing. cancelling the import job.")
		err := c.importer.Cancel(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to cancel repository import")
		}
	}

	if err := c.repoStore.Purge(ctx, repo.ID, repo.Deleted); err != nil {
		return fmt.Errorf("failed to delete repo from db: %w", err)
	}

	if err := c.DeleteGitRepository(ctx, session, repo.GitUID); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to remove git repository")
	}

	c.eventReporter.Deleted(
		ctx,
		&repoevents.DeletedPayload{
			RepoID: repo.ID,
		},
	)
	return nil
}

func (c *Controller) DeleteGitRepository(
	ctx context.Context,
	session *auth.Session,
	gitUID string,
) error {
	// create custom write params for delete as repo might or might not exist in db (similar to create).
	envVars, err := githook.GenerateEnvironmentVariables(
		ctx,
		c.urlProvider.GetInternalAPIURL(ctx),
		0, // no repoID
		session.Principal.ID,
		true,
		true,
	)
	if err != nil {
		return fmt.Errorf("failed to generate git hook environment variables: %w", err)
	}
	writeParams := git.WriteParams{
		Actor: git.Identity{
			Name:  session.Principal.DisplayName,
			Email: session.Principal.Email,
		},
		RepoUID: gitUID,
		EnvVars: envVars,
	}

	err = c.git.DeleteRepository(ctx, &git.DeleteRepositoryParams{
		WriteParams: writeParams,
	})

	// deletion should not fail if repo dir does not exist.
	if errors.IsNotFound(err) {
		log.Ctx(ctx).Warn().Str("repo.git_uid", gitUID).
			Msg("git repository directory does not exist")
	} else if err != nil {
		return fmt.Errorf("failed to remove git repository %s: %w", gitUID, err)
	}
	return nil
}
