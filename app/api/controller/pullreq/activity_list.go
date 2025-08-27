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

// ActivityList returns a list of pull request activities
// from the provided repository and pull request number.
func (c *Controller) ActivityList(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	filter *types.PullReqActivityFilter,
) ([]*types.PullReqActivity, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	list, err := c.activityStore.List(ctx, pr.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests activities: %w", err)
	}

	for _, act := range list {
		if act.Metadata == nil || act.Metadata.Mentions == nil {
			continue
		}

		if act.Metadata.Mentions.IDs != nil {
			mentions, err := c.principalInfoCache.Map(ctx, act.Metadata.Mentions.IDs)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch activity mentions from principalInfoView: %w", err)
			}
			act.Mentions = mentions
		}
		if act.Metadata.Mentions.UserGroupIDs != nil {
			groups, err := c.userGroupStore.Map(ctx, act.Metadata.Mentions.UserGroupIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch activity mentions from userGroupStore: %w", err)
			}
			groupInfoMentions := make(map[int64]*types.UserGroupInfo, len(groups))
			for id, g := range groups {
				if g != nil {
					groupInfoMentions[id] = g.ToUserGroupInfo()
				}
			}
			act.GroupMentions = groupInfoMentions
		}
	}

	list = removeDeletedComments(list)

	return list, nil
}

func allCommentsDeleted(comments []*types.PullReqActivity) bool {
	for _, comment := range comments {
		if comment.Deleted == nil {
			return false
		}
	}
	return true
}

// removeDeletedComments removes all (ordinary comment and change comment) threads
// (the top level comment and all replies to it), but only if all comments
// in the thread are deleted. Just one non-deleted reply in a thread will cause
// the entire thread to be included in the resulting list.
func removeDeletedComments(list []*types.PullReqActivity) []*types.PullReqActivity {
	var (
		threadIdx int
		threadLen int
		listIdx   int
	)

	inspectThread := func() {
		if threadLen > 0 && !allCommentsDeleted(list[threadIdx:threadIdx+threadLen]) {
			copy(list[listIdx:listIdx+threadLen], list[threadIdx:threadIdx+threadLen])
			listIdx += threadLen
		}
		threadLen = 0
	}

	for i, act := range list {
		if act.Deleted != nil {
			act.Text = "" // return deleted comments, but remove their content
		}

		if act.Kind == enum.PullReqActivityKindComment || act.Kind == enum.PullReqActivityKindChangeComment {
			if threadLen == 0 || list[threadIdx].Order != act.Order {
				inspectThread()
				threadIdx = i
				threadLen = 1
			} else {
				threadLen++
			}
			continue
		}

		inspectThread()

		list[listIdx] = act
		listIdx++
	}

	inspectThread()

	return list[:listIdx]
}
