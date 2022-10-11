package gitrpc

import (
	"context"

	"code.gitea.io/gitea/modules/git"
)

type gitea struct {
}

func newGitea() (gitea, error) {
	err := git.InitSimple(context.Background())
	if err != nil {
		return gitea{}, err
	}

	return gitea{}, nil
}

// InitRepository initializes a new Git repository.
func (g gitea) InitRepository(ctx context.Context, path string, bare bool) error {
	return git.InitRepository(ctx, path, bare)
}

// IsRepoURLAccessible checks if given repository URL is accessible.
func (g gitea) IsRepoURLAccessible(ctx context.Context, url string) bool {
	return git.IsRepoURLAccessible(ctx, url)
}

func (g gitea) Clone(ctx context.Context, from, to string, opts cloneRepoOption) error {
	return git.Clone(ctx, from, to, git.CloneRepoOptions{
		Timeout:       opts.timeout,
		Mirror:        opts.mirror,
		Bare:          opts.bare,
		Quiet:         opts.quiet,
		Branch:        opts.branch,
		Shared:        opts.shared,
		NoCheckout:    opts.noCheckout,
		Depth:         opts.depth,
		Filter:        opts.filter,
		SkipTLSVerify: opts.skipTLSVerify,
	})
}

func (g gitea) AddFiles(repoPath string, all bool, files ...string) error {
	return git.AddChanges(repoPath, all, files...)
}

func (g gitea) Commit(repoPath string, opts commitChangesOptions) error {
	return git.CommitChanges(repoPath, git.CommitChangesOptions{
		Committer: &git.Signature{
			Name:  opts.committer.name,
			Email: opts.committer.email,
			When:  opts.committer.when,
		},
		Author: &git.Signature{
			Name:  opts.author.name,
			Email: opts.author.email,
			When:  opts.author.when,
		},
		Message: opts.message,
	})
}

func (g gitea) Push(ctx context.Context, repoPath string, opts pushOptions) error {
	return git.Push(ctx, repoPath, git.PushOptions{
		Remote:  opts.remote,
		Branch:  opts.branch,
		Force:   opts.force,
		Mirror:  opts.mirror,
		Env:     opts.env,
		Timeout: opts.timeout,
	})
}
