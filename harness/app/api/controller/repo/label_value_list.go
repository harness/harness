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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListLabelValues lists all label values defined for the specified repository.
func (c *Controller) ListLabelValues(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	key string,
	filter types.ListQueryFilter,
) ([]*types.LabelValue, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	values, count, err := c.labelSvc.ListValues(ctx, nil, &repo.ID, key, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list repo label values: %w", err)
	}

	return values, count, nil
}
