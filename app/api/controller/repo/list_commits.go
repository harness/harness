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
	"regexp"
	"strings"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/maps"
)

// ListCommits lists the commits of a repo.
func (c *Controller) ListCommits(ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
	filter *types.CommitFilter,
) (types.ListCommitResponse, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return types.ListCommitResponse{}, err
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	committerRegex, err := c.contributorsRegex(ctx, filter.Committer, filter.CommitterIDs)
	if err != nil {
		return types.ListCommitResponse{}, fmt.Errorf("failed create committer regex: %w", err)
	}

	authorRegex, err := c.contributorsRegex(ctx, filter.Author, filter.AuthorIDs)
	if err != nil {
		return types.ListCommitResponse{}, fmt.Errorf("failed create author regex: %w", err)
	}

	rpcOut, err := c.git.ListCommits(ctx, &git.ListCommitsParams{
		ReadParams:   git.CreateReadParams(repo),
		GitREF:       gitRef,
		After:        filter.After,
		Page:         int32(filter.Page),  //nolint:gosec
		Limit:        int32(filter.Limit), //nolint:gosec
		Path:         filter.Path,
		Since:        filter.Since,
		Until:        filter.Until,
		Committer:    committerRegex,
		Author:       authorRegex,
		IncludeStats: filter.IncludeStats,
		Regex:        true,
	})
	if err != nil {
		return types.ListCommitResponse{}, err
	}

	commits := make([]types.Commit, len(rpcOut.Commits))
	for i := range rpcOut.Commits {
		commits[i] = *controller.MapCommit(&rpcOut.Commits[i])
	}

	renameDetailList := make([]types.RenameDetails, len(rpcOut.RenameDetails))
	for i := range rpcOut.RenameDetails {
		renameDetails := controller.MapRenameDetails(rpcOut.RenameDetails[i])
		if renameDetails == nil {
			return types.ListCommitResponse{}, fmt.Errorf("rename details was nil")
		}
		renameDetailList[i] = *renameDetails
	}
	return types.ListCommitResponse{
		Commits:       commits,
		RenameDetails: renameDetailList,
		TotalCommits:  rpcOut.TotalCommits,
	}, nil
}

func (c *Controller) contributorsRegex(
	ctx context.Context,
	identifier string,
	ids []int64,
) (string, error) {
	if identifier == "" && len(ids) == 0 {
		return "", nil
	}

	var emailRegex string
	if len(ids) > 0 {
		principals, err := c.principalInfoCache.Map(ctx, ids)
		if err != nil {
			return "", err
		}
		if len(principals) > 0 {
			parts := make([]string, len(principals))

			for i, principal := range maps.Values(principals) {
				parts[i] = regexp.QuoteMeta(principal.Email)
			}

			emailRegex = "\\<(" + strings.Join(parts, "|") + ")\\>"
		}
	}

	var regex string
	switch {
	case identifier != "" && emailRegex != "":
		regex = regexp.QuoteMeta(identifier) + "|" + emailRegex
	case identifier != "":
		regex = regexp.QuoteMeta(identifier)
	case emailRegex != "":
		regex = emailRegex
	}

	return regex, nil
}
