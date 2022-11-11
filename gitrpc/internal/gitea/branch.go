// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"fmt"
	"strings"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/harness/gitness/gitrpc/internal/types"
)

// CreateBranch creates a new branch.
// Note: target is the commit (or points to the commit) the branch will be pointing to.
func (g Adapter) CreateBranch(ctx context.Context, repoPath string,
	branchName string, target string) (*types.Branch, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	// Get the commit object for the ref
	giteaCommit, err := giteaRepo.GetCommit(target)
	if err != nil {
		return nil, processGiteaErrorf(err, "error getting commit for ref '%s'", target)
	}

	// In case of target being an annotated tag, gitea is overwriting the commit message with the tag message.
	// Reload the commit explicitly in case it's a tag (independent of whether it's annotated or not to simplify code)
	// NOTE: we also allow passing refs/tags/tagName or tags/tagName, which is not covered by IsTagExist.
	// Worst case we have a false positive and reload the same commit, we don't want false negatives though!
	if strings.HasPrefix(target, gitea.TagPrefix) || strings.HasPrefix(target, "tags") || giteaRepo.IsTagExist(target) {
		giteaCommit, err = giteaRepo.GetCommit(giteaCommit.ID.String())
		if err != nil {
			return nil, processGiteaErrorf(err, "error getting commit for annotated tag '%s' (commitId '%s')",
				target, giteaCommit.ID.String())
		}
	}

	err = giteaRepo.CreateBranch(branchName, giteaCommit.ID.String())
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to create branch '%s'", branchName)
	}

	commit, err := mapGiteaCommit(giteaCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to map gitea commit: %w", err)
	}

	return &types.Branch{
		Name:   branchName,
		SHA:    giteaCommit.ID.String(),
		Commit: commit,
	}, nil
}

// DeleteBranch deletes an existing branch.
func (g Adapter) DeleteBranch(ctx context.Context, repoPath string, branchName string, force bool) error {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return err
	}
	defer giteaRepo.Close()

	err = giteaRepo.DeleteBranch(branchName, gitea.DeleteBranchOptions{Force: force})
	if err != nil {
		return processGiteaErrorf(err, "failed to delete branch '%s'", branchName)
	}

	return nil
}
