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

package gitea

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc/internal/types"

	gitea "code.gitea.io/gitea/modules/git"
)

// GetBranch gets an existing branch.
func (g Adapter) GetBranch(ctx context.Context, repoPath string,
	branchName string) (*types.Branch, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to open repository")
	}
	defer giteaRepo.Close()

	giteaBranch, err := giteaRepo.GetBranch(branchName)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to get branch '%s'", branchName)
	}

	giteaCommit, err := giteaBranch.GetCommit()
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to get commit '%s'", branchName)
	}

	commit, err := mapGiteaCommit(giteaCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to map gitea commit: %w", err)
	}

	return &types.Branch{
		Name:   giteaBranch.Name,
		SHA:    giteaCommit.ID.String(),
		Commit: commit,
	}, nil
}
