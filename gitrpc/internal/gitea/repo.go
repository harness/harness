// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"fmt"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/harness/gitness/gitrpc/internal/types"
)

// InitRepository initializes a new Git repository.
func (g Adapter) InitRepository(ctx context.Context, repoPath string, bare bool) error {
	return gitea.InitRepository(ctx, repoPath, bare)
}

// SetDefaultBranch sets the default branch of a repo.
func (g Adapter) SetDefaultBranch(ctx context.Context, repoPath string,
	defaultBranch string, allowEmpty bool) error {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return err
	}
	defer giteaRepo.Close()

	// if requested, error out if branch doesn't exist. Otherwise, blindly set it.
	if !allowEmpty && !giteaRepo.IsBranchExist(defaultBranch) {
		// TODO: ensure this returns not found error to caller
		return fmt.Errorf("branch '%s' does not exist", defaultBranch)
	}

	// change default branch
	err = giteaRepo.SetDefaultBranch(defaultBranch)
	if err != nil {
		return processGiteaErrorf(err, "failed to set new default branch")
	}

	return nil
}

func (g Adapter) Clone(ctx context.Context, from, to string, opts types.CloneRepoOptions) error {
	err := gitea.Clone(ctx, from, to, gitea.CloneRepoOptions{
		Timeout:       opts.Timeout,
		Mirror:        opts.Mirror,
		Bare:          opts.Bare,
		Quiet:         opts.Quiet,
		Branch:        opts.Branch,
		Shared:        opts.Shared,
		NoCheckout:    opts.NoCheckout,
		Depth:         opts.Depth,
		Filter:        opts.Filter,
		SkipTLSVerify: opts.SkipTLSVerify,
	})
	if err != nil {
		return processGiteaErrorf(err, "failed to clone repo")
	}

	return nil
}

func (g Adapter) AddFiles(repoPath string, all bool, files ...string) error {
	err := gitea.AddChanges(repoPath, all, files...)
	if err != nil {
		return processGiteaErrorf(err, "failed to add changes")
	}

	return nil
}

func (g Adapter) Commit(repoPath string, opts types.CommitChangesOptions) error {
	err := gitea.CommitChanges(repoPath, gitea.CommitChangesOptions{
		Committer: &gitea.Signature{
			Name:  opts.Committer.Identity.Name,
			Email: opts.Committer.Identity.Email,
			When:  opts.Committer.When,
		},
		Author: &gitea.Signature{
			Name:  opts.Author.Identity.Name,
			Email: opts.Author.Identity.Email,
			When:  opts.Author.When,
		},
		Message: opts.Message,
	})
	if err != nil {
		return processGiteaErrorf(err, "failed to commit changes")
	}

	return nil
}

func (g Adapter) Push(ctx context.Context, repoPath string, opts types.PushOptions) error {
	err := gitea.Push(ctx, repoPath, gitea.PushOptions{
		Remote:  opts.Remote,
		Branch:  opts.Branch,
		Force:   opts.Force,
		Mirror:  opts.Mirror,
		Env:     opts.Env,
		Timeout: opts.Timeout,
	})
	if err != nil {
		return processGiteaErrorf(err, "failed to push changes")
	}

	return nil
}
