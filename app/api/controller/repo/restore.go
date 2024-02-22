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
	"github.com/harness/gitness/types/enum"
)

type RestoreInput struct {
	NewIdentifier string `json:"new_identifier,omitempty"`
}

func (c *Controller) Restore(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	deletedAt int64,
	in *RestoreInput,
) (*types.Repository, error) {
	repo, err := c.repoStore.FindByRefAndDeletedAt(ctx, repoRef, deletedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	if repo.Deleted == nil {
		return nil, usererror.BadRequest("cannot restore a repo that hasn't been deleted")
	}

	repo, err = c.repoStore.Restore(ctx, repo, in.NewIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to restore the repo: %w", err)
	}

	return repo, nil
}
