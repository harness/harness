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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

// DeleteLabel deletes a label for the specified repository.
func (c *Controller) DeleteLabel(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	key string,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	if err := c.labelSvc.Delete(ctx, nil, &repo.ID, key); err != nil {
		return fmt.Errorf("failed to delete repo label: %w", err)
	}

	return nil
}
