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

package adapter

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/util"
	"github.com/rs/zerolog/log"
)

const (
	gitReferenceNamePrefixBranch = "refs/heads/"
	gitReferenceNamePrefixTag    = "refs/tags/"
)

var lsRemoteHeadRegexp = regexp.MustCompile(`ref: refs/heads/([^\s]+)\s+HEAD`)

// InitRepository initializes a new Git repository.
func (a Adapter) InitRepository(
	ctx context.Context,
	repoPath string,
	bare bool,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	err := os.MkdirAll(repoPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory '%s', err: %w", repoPath, err)
	}

	cmd := command.New("init")
	if bare {
		cmd.Add(command.WithFlag("--bare"))
	}
	return cmd.Run(ctx, command.WithDir(repoPath))
}

func (a Adapter) OpenRepository(
	ctx context.Context,
	repoPath string,
) (*gitea.Repository, error) {
	repo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to open repository")
	}
	return repo, nil
}

// SetDefaultBranch sets the default branch of a repo.
func (a Adapter) SetDefaultBranch(
	ctx context.Context,
	repoPath string,
	defaultBranch string,
	allowEmpty bool,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return processGiteaErrorf(err, "failed to open repository")
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

// GetDefaultBranch gets the default branch of a repo.
func (a Adapter) GetDefaultBranch(
	ctx context.Context,
	repoPath string,
) (string, error) {
	if repoPath == "" {
		return "", ErrRepositoryPathEmpty
	}
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return "", processGiteaErrorf(err, "failed to open gitea repo")
	}
	defer giteaRepo.Close()

	// get default branch
	branch, err := giteaRepo.GetDefaultBranch()
	if err != nil {
		return "", processGiteaErrorf(err, "failed to get default branch")
	}

	return branch, nil
}

// GetRemoteDefaultBranch retrieves the default branch of a remote repository.
// If the repo doesn't have a default branch, types.ErrNoDefaultBranch is returned.
func (a Adapter) GetRemoteDefaultBranch(
	ctx context.Context,
	remoteURL string,
) (string, error) {
	args := []string{
		"-c", "credential.helper=",
		"ls-remote",
		"--symref",
		"-q",
		remoteURL,
		"HEAD",
	}

	cmd := gitea.NewCommand(ctx, args...)
	stdOut, _, err := cmd.RunStdString(nil)
	if err != nil {
		return "", processGiteaErrorf(err, "failed to ls remote repo")
	}

	// git output looks as follows, and we are looking for the ref that HEAD points to
	// 		ref: refs/heads/main    HEAD
	// 		46963bc7f0b5e8c5f039d50ac9e6e51933c78cdf        HEAD
	match := lsRemoteHeadRegexp.FindStringSubmatch(stdOut)
	if match == nil {
		return "", types.ErrNoDefaultBranch
	}

	return match[1], nil
}

func (a Adapter) Clone(
	ctx context.Context,
	from string,
	to string,
	opts types.CloneRepoOptions,
) error {
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

// Sync synchronizes the repository to match the provided source.
// NOTE: This is a read operation and doesn't trigger any server side hooks.
func (a Adapter) Sync(
	ctx context.Context,
	repoPath string,
	source string,
	refSpecs []string,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	if len(refSpecs) == 0 {
		refSpecs = []string{"+refs/*:refs/*"}
	}
	args := []string{
		"-c", "advice.fetchShowForcedUpdates=false",
		"-c", "credential.helper=",
		"fetch",
		"--quiet",
		"--prune",
		"--atomic",
		"--force",
		"--no-write-fetch-head",
		"--no-show-forced-updates",
		source,
	}
	args = append(args, refSpecs...)

	cmd := gitea.NewCommand(ctx, args...)
	_, _, err := cmd.RunStdString(&gitea.RunOpts{
		Dir:               repoPath,
		UseContextTimeout: true,
	})
	if err != nil {
		return processGiteaErrorf(err, "failed to sync repo")
	}

	return nil
}

func (a Adapter) AddFiles(
	repoPath string,
	all bool,
	files ...string,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	err := gitea.AddChanges(repoPath, all, files...)
	if err != nil {
		return processGiteaErrorf(err, "failed to add changes")
	}

	return nil
}

// Commit commits the changes of the repository.
// NOTE: Modification of gitea implementation that supports commiter_date + author_date.
func (a Adapter) Commit(
	ctx context.Context,
	repoPath string,
	opts types.CommitChangesOptions,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}

	cmd := command.New("commit",
		command.WithFlag("-m", opts.Message),
		command.WithAuthorAndDate(
			opts.Author.Identity.Name,
			opts.Author.Identity.Email,
			opts.Author.When,
		),
		command.WithCommitterAndDate(
			opts.Committer.Identity.Name,
			opts.Committer.Identity.Email,
			opts.Committer.When,
		),
	)
	err := cmd.Run(ctx, command.WithDir(repoPath))
	// No stderr but exit status 1 means nothing to commit (see gitea CommitChanges)
	if err != nil && err.Error() != "exit status 1" {
		return processGiteaErrorf(err, "failed to commit changes")
	}
	return nil
}

// Push pushs local commits to given remote branch.
// NOTE: Modification of gitea implementation that supports --force-with-lease.
// TODOD: return our own error types and move to above adapter.Push method
func (a Adapter) Push(
	ctx context.Context,
	repoPath string,
	opts types.PushOptions,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	cmd := gitea.NewCommand(ctx,
		"-c", "credential.helper=",
		"push",
	)
	if opts.Force {
		cmd.AddArguments("-f")
	}
	if opts.ForceWithLease != "" {
		cmd.AddArguments(fmt.Sprintf("--force-with-lease=%s", opts.ForceWithLease))
	}
	if opts.Mirror {
		cmd.AddArguments("--mirror")
	}
	cmd.AddArguments("--", opts.Remote)

	if len(opts.Branch) > 0 {
		cmd.AddArguments(opts.Branch)
	}

	// remove credentials if there are any
	if strings.Contains(opts.Remote, "://") && strings.Contains(opts.Remote, "@") {
		opts.Remote = util.SanitizeCredentialURLs(opts.Remote)
	}

	if opts.Timeout == 0 {
		opts.Timeout = -1
	}

	if a.traceGit {
		// create copy to not modify original underlying array
		opts.Env = append([]string{gitTrace + "=true"}, opts.Env...)
	}

	cmd.SetDescription(
		fmt.Sprintf(
			"pushing %s to %s (Force: %t, ForceWithLease: %s)",
			opts.Branch,
			opts.Remote,
			opts.Force,
			opts.ForceWithLease,
		),
	)

	var outbuf, errbuf strings.Builder
	err := cmd.Run(&gitea.RunOpts{
		Env:     opts.Env,
		Timeout: opts.Timeout,
		Dir:     repoPath,
		Stdout:  &outbuf,
		Stderr:  &errbuf,
	})

	if a.traceGit {
		log.Ctx(ctx).Trace().
			Str("git", "push").
			Err(err).
			Msgf("IN:\n%#v\n\nSTDOUT:\n%s\n\nSTDERR:\n%s", opts, outbuf.String(), errbuf.String())
	}

	if err != nil {
		switch {
		case strings.Contains(errbuf.String(), "non-fast-forward"):
			return &gitea.ErrPushOutOfDate{
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
		case strings.Contains(errbuf.String(), "! [remote rejected]"):
			err := &gitea.ErrPushRejected{
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
			err.GenerateMessage()
			return err
		case strings.Contains(errbuf.String(), "matches more than one"):
			err := &gitea.ErrMoreThanOne{
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
			return err
		default:
			// fall through to normal error handling
		}
	}

	if err != nil {
		// add commandline error output to error
		if errbuf.Len() > 0 {
			err = fmt.Errorf("%w\ncmd error output: %s", err, errbuf.String())
		}

		return processGiteaErrorf(err, "failed to push changes")
	}

	return nil
}

func (a Adapter) CountObjects(ctx context.Context, repoPath string) (types.ObjectCount, error) {
	cmd := gitea.NewCommand(ctx,
		"count-objects", "-v",
	)

	var outbuf strings.Builder
	if err := cmd.Run(&gitea.RunOpts{
		Dir:    repoPath,
		Stdout: &outbuf,
	}); err != nil {
		return types.ObjectCount{}, fmt.Errorf("error running git count-objects: %w", err)
	}

	objectCount := parseGitCountObjectsOutput(ctx, outbuf.String())
	return objectCount, nil
}

func parseGitCountObjectsOutput(ctx context.Context, output string) types.ObjectCount {
	info := types.ObjectCount{}

	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Fields(line)

		switch fields[0] {
		case "count:":
			fmt.Sscanf(fields[1], "%d", &info.Count)
		case "size:":
			fmt.Sscanf(fields[1], "%d", &info.Size)
		case "in-pack:":
			fmt.Sscanf(fields[1], "%d", &info.InPack)
		case "packs:":
			fmt.Sscanf(fields[1], "%d", &info.Packs)
		case "size-pack:":
			fmt.Sscanf(fields[1], "%d", &info.SizePack)
		case "prune-packable:":
			fmt.Sscanf(fields[1], "%d", &info.PrunePackable)
		case "garbage:":
			fmt.Sscanf(fields[1], "%d", &info.Garbage)
		case "size-garbage:":
			fmt.Sscanf(fields[1], "%d", &info.SizeGarbage)
		default:
			log.Ctx(ctx).Warn().Msgf("line '%s: %s' not processed", fields[0], fields[1])
		}
	}

	return info
}
