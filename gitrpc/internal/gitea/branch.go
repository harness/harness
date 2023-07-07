// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
