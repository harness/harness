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

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListLabels list labels assigned to a specified pullreq.
func (c *Controller) ListLabels(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	filter *types.AssignableLabelFilter,
) (*types.ScopesLabels, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pullreq, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find pullreq: %w", err)
	}

	scopeLabelsMap, total, err := c.labelSvc.ListPullReqLabels(
		ctx, repo, repo.ParentID, pullreq.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pullreq labels: %w", err)
	}

	return scopeLabelsMap, total, nil
}
