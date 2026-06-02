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

type PullReqViewResponse struct {
	Groups []*types.PullReqFileGroupWithFiles `json:"groups"`
}

// PullReqViewGet returns all pull request file groups together with their files.
func (c *Controller) PullReqViewGet(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) (*PullReqViewResponse, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	groups, err := c.fileGroupStore.List(ctx, pr.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull request file groups: %w", err)
	}

	filteredGroups := make([]*types.PullReqFileGroupWithFiles, 0, len(groups))
	for _, group := range groups {
		if len(group.Files) == 0 {
			continue
		}
		filteredGroups = append(filteredGroups, group)
	}

	return &PullReqViewResponse{Groups: filteredGroups}, nil
}
