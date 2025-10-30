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
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type PathsDetailsInput struct {
	Paths []string `json:"paths"`
}

type PathsDetailsOutput struct {
	Details []types.PathDetails `json:"details"`
}

// PathsDetails finds the additional info about the provided paths of the repo.
// If no gitRef is provided, the content is retrieved from the default branch.
func (c *Controller) PathsDetails(ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
	input PathsDetailsInput,
) (PathsDetailsOutput, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return PathsDetailsOutput{}, err
	}

	if len(input.Paths) == 0 {
		return PathsDetailsOutput{}, nil
	}

	const maxInputPaths = 50
	if len(input.Paths) > maxInputPaths {
		return PathsDetailsOutput{},
			usererror.BadRequestf("Maximum number of elements in the Paths array is %d", maxInputPaths)
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	result, err := c.git.PathsDetails(ctx, git.PathsDetailsParams{
		ReadParams: git.CreateReadParams(repo),
		GitREF:     gitRef,
		Paths:      input.Paths,
	})
	if err != nil {
		return PathsDetailsOutput{}, err
	}

	commits := make([]*types.Commit, 0, len(result.Details))
	output := PathsDetailsOutput{
		Details: make([]types.PathDetails, len(result.Details)),
	}
	for i, d := range result.Details {
		lastCommit := controller.MapCommit(d.LastCommit)
		output.Details[i] = types.PathDetails{
			Path:       d.Path,
			LastCommit: lastCommit,
		}
		if lastCommit != nil {
			commits = append(commits, lastCommit)
		}
	}

	err = c.signatureVerifyService.VerifyCommits(ctx, repo.ID, commits)
	if err != nil {
		return PathsDetailsOutput{}, fmt.Errorf("failed to verify signature of commits: %w", err)
	}

	return output, nil
}
