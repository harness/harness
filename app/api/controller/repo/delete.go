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
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/gitrpc"
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

	if repo.Importing {
		err = c.importer.Cancel(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to cancel repository import")
		}

		return c.DeleteNoAuth(ctx, session, repo)
	}

	log.Ctx(ctx).Info().Msgf("Delete request received for repo %s , id: %d", repo.Path, repo.ID)

	return c.DeleteNoAuth(ctx, session, repo)
}

func (c *Controller) DeleteNoAuth(ctx context.Context, session *auth.Session, repo *types.Repository) error {
	if err := c.deleteGitRPCRepository(ctx, session, repo); err != nil {
		return fmt.Errorf("failed to delete git repository: %w", err)
	}

	if err := c.repoStore.Delete(ctx, repo.ID); err != nil {
		return fmt.Errorf("failed to delete repo from db: %w", err)
	}

	return nil
}

func (c *Controller) deleteGitRPCRepository(
	ctx context.Context,
	session *auth.Session,
	repo *types.Repository,
) error {
	writeParams, err := CreateRPCWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return fmt.Errorf("failed to create RPC write params: %w", err)
	}

	err = c.gitRPCClient.DeleteRepository(ctx, &gitrpc.DeleteRepositoryParams{
		WriteParams: writeParams,
	})

	// deletion should not fail if dir does not exist in repos dir
	if gitrpc.ErrorStatus(err) == gitrpc.StatusNotFound {
		log.Ctx(ctx).Warn().Msgf("gitrpc repo %s does not exist", repo.GitUID)
	} else if err != nil {
		// deletion has failed before removing(rename) the repo dir
		return fmt.Errorf("gitrpc failed to delete repo %s: %w", repo.GitUID, err)
	}
	return nil
}
