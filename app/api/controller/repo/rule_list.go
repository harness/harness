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
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RuleList returns protection rules for a repository.
func (c *Controller) RuleList(ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.RuleFilter,
) ([]types.Rule, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, err
	}

	var list []types.Rule
	var count int64

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		list, err = c.ruleStore.List(ctx, nil, &repo.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list repository-level protection rules: %w", err)
		}

		if filter.Page == 1 && len(list) < filter.Size {
			count = int64(len(list))
			return nil
		}

		count, err = c.ruleStore.Count(ctx, nil, &repo.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count repository-level protection rules: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	for i := range list {
		list[i].Users, err = c.getRuleUsers(ctx, &list[i])
		if err != nil {
			return nil, 0, err
		}
	}

	return list, count, nil
}
