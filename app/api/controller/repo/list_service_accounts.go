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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListServiceAccounts lists the service accounts of a repo.
func (c *Controller) ListServiceAccounts(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) ([]*types.ServiceAccount, error) {
	repo, err := c.getRepo(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	if err := apiauth.CheckServiceAccount(
		ctx,
		c.authorizer,
		session,
		c.spaceStore,
		c.repoStore,
		enum.ParentResourceTypeRepo,
		repo.ID,
		"",
		enum.PermissionServiceAccountView,
	); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return c.principalStore.ListServiceAccounts(ctx, enum.ParentResourceTypeRepo, repo.ID)
}
