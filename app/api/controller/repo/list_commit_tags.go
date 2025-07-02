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

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListCommitTags lists the commit tags of a repo.
func (c *Controller) ListCommitTags(ctx context.Context,
	session *auth.Session,
	repoRef string,
	includeCommit bool,
	filter *types.TagFilter,
) ([]*types.CommitTag, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	result, err := c.git.ListCommitTags(ctx, &git.ListCommitTagsParams{
		ReadParams:    git.CreateReadParams(repo),
		IncludeCommit: includeCommit,
		Query:         filter.Query,
		Sort:          mapToRPCTagSortOption(filter.Sort),
		Order:         mapToRPCSortOrder(filter.Order),
		Page:          int32(filter.Page), //nolint:gosec
		PageSize:      int32(filter.Size), //nolint:gosec
	})
	if err != nil {
		return nil, err
	}

	tags := make([]*types.CommitTag, len(result.Tags))
	for i := range result.Tags {
		t := controller.MapCommitTag(result.Tags[i])
		tags[i] = &t
	}

	verifySession := c.signatureVerifyService.NewVerifySession(repo.ID)

	err = verifySession.VerifyCommitTags(ctx, tags)
	if err != nil {
		return nil, fmt.Errorf("failed to verify tags: %w", err)
	}

	commits := make([]*types.Commit, 0, len(tags))
	for _, tag := range tags {
		if tag.Commit != nil {
			commits = append(commits, tag.Commit)
		}
	}

	err = verifySession.VerifyCommits(ctx, commits)
	if err != nil {
		return nil, fmt.Errorf("failed to verify signature of tags' commits: %w", err)
	}

	verifySession.StoreSignatures(ctx)

	return tags, nil
}

func mapToRPCTagSortOption(o enum.TagSortOption) git.TagSortOption {
	switch o {
	case enum.TagSortOptionDate:
		return git.TagSortOptionDate
	case enum.TagSortOptionName:
		return git.TagSortOptionName
	case enum.TagSortOptionDefault:
		return git.TagSortOptionDefault
	default:
		// no need to error out - just use default for sorting
		return git.TagSortOptionDefault
	}
}
