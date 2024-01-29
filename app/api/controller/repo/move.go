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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// MoveInput is used for moving a repo.
type MoveInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        *string `json:"uid" deprecated:"true"`
	Identifier *string `json:"identifier"`
}

func (i *MoveInput) hasChanges(repo *types.Repository) bool {
	if i.Identifier != nil && *i.Identifier != repo.Identifier {
		return true
	}

	return false
}

// Move moves a repository to a new identifier.
// TODO: Add support for moving to other parents and aliases.
//
//nolint:gocognit // refactor if needed
func (c *Controller) Move(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *MoveInput,
) (*types.Repository, error) {
	if err := c.sanitizeMoveInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if repo.Importing {
		return nil, usererror.BadRequest("can't move a repo that is being imported")
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	if !in.hasChanges(repo) {
		return repo, nil
	}

	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		if in.Identifier != nil {
			r.Identifier = *in.Identifier
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update repo: %w", err)
	}

	repo.GitURL = c.urlProvider.GenerateGITCloneURL(repo.Path)

	return repo, nil
}

func (c *Controller) sanitizeMoveInput(in *MoveInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == nil {
		in.Identifier = in.UID
	}

	if in.Identifier != nil {
		if err := check.RepoIdentifier(*in.Identifier); err != nil {
			return err
		}
	}

	return nil
}
