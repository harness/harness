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

package api

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"

	"github.com/rs/zerolog/log"
)

type CloneRepoOptions struct {
	Timeout       time.Duration
	Mirror        bool
	Bare          bool
	Quiet         bool
	Branch        string
	Shared        bool
	NoCheckout    bool
	Depth         int
	Filter        string
	SkipTLSVerify bool
}

type PushOptions struct {
	Remote         string
	Branch         string
	Force          bool
	ForceWithLease string
	Env            []string
	Timeout        time.Duration
	Mirror         bool
}

// ObjectCount represents the parsed information from the `git count-objects -v` command.
// For field meanings, see https://git-scm.com/docs/git-count-objects#_options.
type ObjectCount struct {
	Count         int
	Size          int64
	InPack        int
	Packs         int
	SizePack      int64
	PrunePackable int
	Garbage       int
	SizeGarbage   int64
}

const (
	gitReferenceNamePrefixBranch = "refs/heads/"
	gitReferenceNamePrefixTag    = "refs/tags/"
)

var lsRemoteHeadRegexp = regexp.MustCompile(`ref: refs/heads/([^\s]+)\s+HEAD`)

// InitRepository initializes a new Git repository.
func (g *Git) InitRepository(
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

// SetDefaultBranch sets the default branch of a repo.
func (g *Git) SetDefaultBranch(
	ctx context.Context,
	repoPath string,
	defaultBranch string,
	ignoreBranchExistance bool,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}

	if !ignoreBranchExistance {
		// best effort try to check for existence - technically someone else could delete it in the meanwhile.
		exist, err := g.IsBranchExist(ctx, repoPath, defaultBranch)
		if err != nil {
			return fmt.Errorf("failed to check if branch exists: %w", err)
		}
		if !exist {
			return errors.NotFoundf("branch %q does not exist", defaultBranch)
		}
	}

	// change default branch
	cmd := command.New("symbolic-ref",
		command.WithArg("HEAD", gitReferenceNamePrefixBranch+defaultBranch),
	)
	err := cmd.Run(ctx, command.WithDir(repoPath))
	if err != nil {
		return processGitErrorf(err, "failed to set new default branch")
	}

	return nil
}

// GetDefaultBranch gets the default branch of a repo.
func (g *Git) GetDefaultBranch(
	ctx context.Context,
	repoPath string,
) (string, error) {
	rawBranchRef, err := g.GetSymbolicRefHeadRaw(ctx, repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get raw symbolic ref HEAD: %w", err)
	}

	branchName := strings.TrimPrefix(
		strings.TrimSpace(
			rawBranchRef,
		),
		BranchPrefix,
	)

	return branchName, nil
}

// GetSymbolicRefHeadRaw returns the raw output of the symolic-ref command for HEAD.
func (g *Git) GetSymbolicRefHeadRaw(
	ctx context.Context,
	repoPath string,
) (string, error) {
	if repoPath == "" {
		return "", ErrRepositoryPathEmpty
	}

	// get default branch
	cmd := command.New("symbolic-ref",
		command.WithArg("HEAD"),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(output))
	if err != nil {
		return "", processGitErrorf(err, "failed to get value of symbolic ref HEAD from git")
	}

	return output.String(), nil
}

// GetRemoteDefaultBranch retrieves the default branch of a remote repository.
// If the repo doesn't have a default branch, types.ErrNoDefaultBranch is returned.
func (g *Git) GetRemoteDefaultBranch(
	ctx context.Context,
	remoteURL string,
) (string, error) {
	cmd := command.New("ls-remote",
		command.WithConfig("credential.helper", ""),
		command.WithFlag("--symref"),
		command.WithFlag("-q"),
		command.WithArg(remoteURL),
		command.WithArg("HEAD"),
	)
	output := &bytes.Buffer{}
	if err := cmd.Run(ctx, command.WithStdout(output)); err != nil {
		return "", processGitErrorf(err, "failed to ls remote repo")
	}

	// git output looks as follows, and we are looking for the ref that HEAD points to
	// 		ref: refs/heads/main    HEAD
	// 		46963bc7f0b5e8c5f039d50ac9e6e51933c78cdf        HEAD
	match := lsRemoteHeadRegexp.FindStringSubmatch(strings.TrimSpace(output.String()))
	if match == nil {
		return "", ErrNoDefaultBranch
	}

	return match[1], nil
}

func (g *Git) Clone(
	ctx context.Context,
	from string,
	to string,
	opts CloneRepoOptions,
) error {
	if err := os.MkdirAll(to, os.ModePerm); err != nil {
		return err
	}

	cmd := command.New("clone")
	if opts.SkipTLSVerify {
		cmd.Add(command.WithConfig("http.sslVerify", "false"))
	}
	if opts.Mirror {
		cmd.Add(command.WithFlag("--mirror"))
	}
	if opts.Bare {
		cmd.Add(command.WithFlag("--bare"))
	}
	if opts.Quiet {
		cmd.Add(command.WithFlag("--quiet"))
	}
	if opts.Shared {
		cmd.Add(command.WithFlag("-s"))
	}
	if opts.NoCheckout {
		cmd.Add(command.WithFlag("--no-checkout"))
	}
	if opts.Depth > 0 {
		cmd.Add(command.WithFlag("--depth", strconv.Itoa(opts.Depth)))
	}
	if opts.Filter != "" {
		cmd.Add(command.WithFlag("--filter", opts.Filter))
	}
	if len(opts.Branch) > 0 {
		cmd.Add(command.WithFlag("-b", opts.Branch))
	}
	cmd.Add(command.WithPostSepArg(from, to))

	if err := cmd.Run(ctx); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// Sync synchronizes the repository to match the provided source.
// NOTE: This is a read operation and doesn't trigger any server side hooks.
func (g *Git) Sync(
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
	cmd := command.New("fetch",
		command.WithConfig("advice.fetchShowForcedUpdates", "false"),
		command.WithConfig("credential.helper", ""),
		command.WithFlag(
			"--quiet",
			"--prune",
			"--atomic",
			"--force",
			"--no-write-fetch-head",
			"--no-show-forced-updates",
		),
		command.WithArg(source),
		command.WithArg(refSpecs...),
	)

	err := cmd.Run(ctx, command.WithDir(repoPath))
	if err != nil {
		return processGitErrorf(err, "failed to sync repo")
	}

	return nil
}

// FetchObjects pull git objects from a different repository.
// It doesn't update any references.
func (g *Git) FetchObjects(
	ctx context.Context,
	repoPath string,
	source string,
	objectSHAs []sha.SHA,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	cmd := command.New("fetch",
		command.WithConfig("advice.fetchShowForcedUpdates", "false"),
		command.WithConfig("credential.helper", ""),
		command.WithFlag(
			"--quiet",
			"--no-auto-gc", // because we're fetching objects that are not referenced
			"--no-tags",
			"--no-write-fetch-head",
			"--no-show-forced-updates",
		),
		command.WithArg(source),
	)

	for _, objectSHA := range objectSHAs {
		cmd.Add(command.WithArg(objectSHA.String()))
	}

	err := cmd.Run(ctx, command.WithDir(repoPath))
	if err != nil {
		if parts := reNotOurRef.FindStringSubmatch(strings.TrimSpace(err.Error())); parts != nil {
			return errors.InvalidArgumentf("Unrecognized git object: %s", parts[1])
		}
		return processGitErrorf(err, "failed to fetch objects")
	}

	return nil
}

// ListRemoteReferences lists references from a remote repository.
func (g *Git) ListRemoteReferences(
	ctx context.Context,
	repoPath string,
	remote string,
	refs ...string,
) (map[string]sha.SHA, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}

	cmd := command.New("ls-remote",
		command.WithFlag("--refs"),
		command.WithArg(remote),
		command.WithPostSepArg(refs...),
	)

	stdout := bytes.NewBuffer(nil)

	if err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(stdout)); err != nil {
		return nil, fmt.Errorf("failed to list references from remote: %w", err)
	}

	result, err := parser.ReferenceList(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse references from remote: %w", err)
	}

	return result, nil
}

// ListLocalReferences lists references from the local repository.
func (g *Git) ListLocalReferences(
	ctx context.Context,
	repoPath string,
	refs ...string,
) (map[string]sha.SHA, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}

	cmd := command.New("show-ref",
		command.WithPostSepArg(refs...),
	)

	stdout := bytes.NewBuffer(nil)

	if err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(stdout)); err != nil {
		return nil, fmt.Errorf("failed to list references: %w", err)
	}

	result, err := parser.ReferenceList(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse references: %w", err)
	}

	return result, nil
}

var reNotOurRef = regexp.MustCompile("upload-pack: not our ref ([a-fA-f0-9]+)$")

func (g *Git) AddFiles(
	ctx context.Context,
	repoPath string,
	all bool,
	files ...string,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}

	cmd := command.New("add")
	if all {
		cmd.Add(command.WithFlag("--all"))
	}
	cmd.Add(command.WithPostSepArg(files...))

	err := cmd.Run(ctx, command.WithDir(repoPath))
	if err != nil {
		return processGitErrorf(err, "failed to add changes")
	}

	return nil
}

// Commit commits the changes of the repository.
func (g *Git) Commit(
	ctx context.Context,
	repoPath string,
	opts CommitChangesOptions,
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
		return processGitErrorf(err, "failed to commit changes")
	}
	return nil
}

// Push pushs local commits to given remote branch.
// TODOD: return our own error types and move to above api.Push method.
func (g *Git) Push(
	ctx context.Context,
	repoPath string,
	opts PushOptions,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	cmd := command.New("push",
		command.WithConfig("credential.helper", ""),
	)
	if opts.Force {
		cmd.Add(command.WithFlag("-f"))
	}
	if opts.ForceWithLease != "" {
		cmd.Add(command.WithFlag("--force-with-lease=" + opts.ForceWithLease))
	}
	if opts.Mirror {
		cmd.Add(command.WithFlag("--mirror"))
	}
	cmd.Add(command.WithPostSepArg(opts.Remote))

	if len(opts.Branch) > 0 {
		cmd.Add(command.WithPostSepArg(opts.Branch))
	}

	if g.traceGit {
		cmd.Add(command.WithEnv(command.GitTrace, "true"))
	}

	// remove credentials if there are any
	if strings.Contains(opts.Remote, "://") && strings.Contains(opts.Remote, "@") {
		opts.Remote = SanitizeCredentialURLs(opts.Remote)
	}

	var outbuf, errbuf strings.Builder
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(&outbuf),
		command.WithStderr(&errbuf),
		command.WithEnvs(opts.Env...),
	)

	if g.traceGit {
		log.Ctx(ctx).Trace().
			Str("git", "push").
			Err(err).
			Msgf("IN:\n%#v\n\nSTDOUT:\n%s\n\nSTDERR:\n%s", opts, outbuf.String(), errbuf.String())
	}

	if err != nil {
		switch {
		case strings.Contains(errbuf.String(), "non-fast-forward"):
			return &PushOutOfDateError{
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
		case strings.Contains(errbuf.String(), "! [remote rejected]"):
			err := &PushRejectedError{
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
			err.GenerateMessage()
			return err
		case strings.Contains(errbuf.String(), "matches more than one"):
			err := &MoreThanOneError{
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

		return processGitErrorf(err, "failed to push changes")
	}

	return nil
}

func (g *Git) CountObjects(ctx context.Context, repoPath string) (ObjectCount, error) {
	var outbuf strings.Builder
	cmd := command.New("count-objects", command.WithFlag("-v"))
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(&outbuf),
	)
	if err != nil {
		return ObjectCount{}, fmt.Errorf("error running git count-objects: %w", err)
	}

	objectCount := parseGitCountObjectsOutput(ctx, outbuf.String())
	return objectCount, nil
}

//nolint:errcheck
func parseGitCountObjectsOutput(ctx context.Context, output string) ObjectCount {
	info := ObjectCount{}

	output = strings.TrimSpace(output)
	lines := strings.SplitSeq(output, "\n")

	for line := range lines {
		fields := strings.Fields(line)

		switch fields[0] {
		case "count:":
			fmt.Sscanf(fields[1], "%d", &info.Count) //nolint:errcheck
		case "size:":
			fmt.Sscanf(fields[1], "%d", &info.Size) //nolint:errcheck
		case "in-pack:":
			fmt.Sscanf(fields[1], "%d", &info.InPack) //nolint:errcheck
		case "packs:":
			fmt.Sscanf(fields[1], "%d", &info.Packs) //nolint:errcheck
		case "size-pack:":
			fmt.Sscanf(fields[1], "%d", &info.SizePack) //nolint:errcheck
		case "prune-packable:":
			fmt.Sscanf(fields[1], "%d", &info.PrunePackable) //nolint:errcheck
		case "garbage:":
			fmt.Sscanf(fields[1], "%d", &info.Garbage) //nolint:errcheck
		case "size-garbage:":
			fmt.Sscanf(fields[1], "%d", &info.SizeGarbage)
		default:
			log.Ctx(ctx).Warn().Msgf("line '%s: %s' not processed", fields[0], fields[1])
		}
	}

	return info
}
