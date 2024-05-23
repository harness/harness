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
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types/enum"
)

type ListPathsOutput struct {
	Files       []string `json:"files,omitempty"`
	Directories []string `json:"directories,omitempty"`
}

// ListPaths lists the paths in the repo for a specific revision.
func (c *Controller) ListPaths(ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
	includeDirectories bool,
) (ListPathsOutput, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return ListPathsOutput{}, err
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	rpcOut, err := c.git.ListPaths(ctx, &git.ListPathsParams{
		ReadParams:         git.CreateReadParams(repo),
		GitREF:             gitRef,
		IncludeDirectories: includeDirectories,
	})
	if err != nil {
		return ListPathsOutput{}, fmt.Errorf("failed to list git paths: %w", err)
	}

	return ListPathsOutput{
		Files:       rpcOut.Files,
		Directories: rpcOut.Directories,
	}, nil
}
