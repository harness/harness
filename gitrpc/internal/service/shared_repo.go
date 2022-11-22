// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
	"github.com/harness/gitness/gitrpc/internal/tempdir"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/rs/zerolog/log"
)

// SharedRepo is a type to wrap our upload repositories as a shallow clone.
type SharedRepo struct {
	repoUID     string
	repo        *git.Repository
	remoteRepo  *git.Repository
	TempBaseDir string
	basePath    string
}

// NewSharedRepo creates a new temporary upload repository.
func NewSharedRepo(tempDir, repoUID string, remoteRepo *git.Repository) (*SharedRepo, error) {
	basePath, err := tempdir.CreateTemporaryPath(tempDir, repoUID)
	if err != nil {
		return nil, err
	}
	t := &SharedRepo{
		repoUID:    repoUID,
		remoteRepo: remoteRepo,
		basePath:   basePath,
	}
	return t, nil
}

// Close the repository cleaning up all files.
func (r *SharedRepo) Close() {
	defer r.repo.Close()
	if err := tempdir.RemoveTemporaryPath(r.basePath); err != nil {
		log.Err(err).Msgf("Failed to remove temporary path %s", r.basePath)
	}
}

// Clone the base repository to our path and set branch as the HEAD.
func (r *SharedRepo) Clone(ctx context.Context, branch string) error {
	if _, _, err := git.NewCommand(ctx, "clone", "-s", "--bare", "-b",
		branch, r.remoteRepo.Path, r.basePath).RunStdString(nil); err != nil {
		stderr := err.Error()
		if matched, _ := regexp.MatchString(".*Remote branch .* not found in upstream origin.*", stderr); matched {
			return git.ErrBranchNotExist{
				Name: branch,
			}
		} else if matched, _ = regexp.MatchString(".* repository .* does not exist.*", stderr); matched {
			return fmt.Errorf("%s %w", r.repoUID, types.ErrNotFound)
		} else {
			return fmt.Errorf("Clone: %w %s", err, stderr)
		}
	}
	gitRepo, err := git.OpenRepository(ctx, r.basePath)
	if err != nil {
		return err
	}
	r.repo = gitRepo
	return nil
}

// Init the repository.
func (r *SharedRepo) Init(ctx context.Context) error {
	if err := git.InitRepository(ctx, r.basePath, false); err != nil {
		return err
	}
	gitRepo, err := git.OpenRepository(ctx, r.basePath)
	if err != nil {
		return err
	}
	r.repo = gitRepo
	return nil
}

// SetDefaultIndex sets the git index to our HEAD.
func (r *SharedRepo) SetDefaultIndex(ctx context.Context) error {
	if _, _, err := git.NewCommand(ctx, "read-tree", "HEAD").RunStdString(&git.RunOpts{Dir: r.basePath}); err != nil {
		return fmt.Errorf("SetDefaultIndex: %w", err)
	}
	return nil
}

// LsFiles checks if the given filename arguments are in the index.
func (r *SharedRepo) LsFiles(ctx context.Context, filenames ...string) ([]string, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)

	cmdArgs := []string{"ls-files", "-z", "--"}
	for _, arg := range filenames {
		if arg != "" {
			cmdArgs = append(cmdArgs, arg)
		}
	}

	if err := git.NewCommand(ctx, cmdArgs...).
		Run(&git.RunOpts{
			Dir:    r.basePath,
			Stdout: stdOut,
			Stderr: stdErr,
		}); err != nil {
		return nil, fmt.Errorf("unable to run git ls-files for temporary repo of: "+
			"%s Error: %w\nstdout: %s\nstderr: %s",
			r.repoUID, err, stdOut.String(), stdErr.String())
	}

	filelist := make([]string, 0)
	for _, line := range bytes.Split(stdOut.Bytes(), []byte{'\000'}) {
		filelist = append(filelist, string(line))
	}

	return filelist, nil
}

// RemoveFilesFromIndex removes the given files from the index.
func (r *SharedRepo) RemoveFilesFromIndex(ctx context.Context, filenames ...string) error {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)
	stdIn := new(bytes.Buffer)
	for _, file := range filenames {
		if file != "" {
			stdIn.WriteString("0 0000000000000000000000000000000000000000\t")
			stdIn.WriteString(file)
			stdIn.WriteByte('\000')
		}
	}

	if err := git.NewCommand(ctx, "update-index", "--remove", "-z", "--index-info").
		Run(&git.RunOpts{
			Dir:    r.basePath,
			Stdin:  stdIn,
			Stdout: stdOut,
			Stderr: stdErr,
		}); err != nil {
		return fmt.Errorf("unable to update-index for temporary repo: %s Error: %w\nstdout: %s\nstderr: %s",
			r.repoUID, err, stdOut.String(), stdErr.String())
	}
	return nil
}

// HashObject writes the provided content to the object db and returns its hash.
func (r *SharedRepo) HashObject(ctx context.Context, content io.Reader) (string, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)

	if err := git.NewCommand(ctx, "hash-object", "-w", "--stdin").
		Run(&git.RunOpts{
			Dir:    r.basePath,
			Stdin:  content,
			Stdout: stdOut,
			Stderr: stdErr,
		}); err != nil {
		return "", fmt.Errorf("unable to hash-object to temporary repo: %s Error: %w\nstdout: %s\nstderr: %s",
			r.repoUID, err, stdOut.String(), stdErr.String())
	}

	return strings.TrimSpace(stdOut.String()), nil
}

// ShowFile dumps show file and write to io.Writer.
func (r *SharedRepo) ShowFile(ctx context.Context, filePath, commitHash string, writer io.Writer) error {
	stderr := new(bytes.Buffer)
	file := strings.TrimSpace(commitHash) + ":" + strings.TrimSpace(filePath)
	cmd := git.NewCommand(ctx, "show", file)
	if err := cmd.Run(&git.RunOpts{
		Dir:    r.repo.Path,
		Stdout: writer,
		Stderr: stderr,
	}); err != nil {
		return fmt.Errorf("show file: %w - %s", err, stderr)
	}
	return nil
}

// AddObjectToIndex adds the provided object hash to the index with the provided mode and path.
func (r *SharedRepo) AddObjectToIndex(ctx context.Context, mode, objectHash, objectPath string) error {
	if _, _, err := git.NewCommand(ctx, "update-index", "--add", "--replace", "--cacheinfo", mode, objectHash,
		objectPath).RunStdString(&git.RunOpts{Dir: r.basePath}); err != nil {
		if matched, _ := regexp.MatchString(".*Invalid path '.*", err.Error()); matched {
			return types.ErrInvalidPath
		}
		return fmt.Errorf("unable to add object to index at %s in temporary repo %s Error: %w",
			objectPath, r.repoUID, err)
	}
	return nil
}

// WriteTree writes the current index as a tree to the object db and returns its hash.
func (r *SharedRepo) WriteTree(ctx context.Context) (string, error) {
	stdout, _, err := git.NewCommand(ctx, "write-tree").RunStdString(&git.RunOpts{Dir: r.basePath})
	if err != nil {
		return "", fmt.Errorf("unable to write-tree in temporary repo for: %s Error: %w",
			r.repoUID, err)
	}
	return strings.TrimSpace(stdout), nil
}

// GetLastCommit gets the last commit ID SHA of the repo.
func (r *SharedRepo) GetLastCommit(ctx context.Context) (string, error) {
	return r.GetLastCommitByRef(ctx, "HEAD")
}

// GetLastCommitByRef gets the last commit ID SHA of the repo by ref.
func (r *SharedRepo) GetLastCommitByRef(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}
	stdout, _, err := git.NewCommand(ctx, "rev-parse", ref).RunStdString(&git.RunOpts{Dir: r.basePath})
	if err != nil {
		return "", fmt.Errorf("unable to rev-parse %s in temporary repo for: %s Error: %w",
			ref, r.repoUID, err)
	}
	return strings.TrimSpace(stdout), nil
}

// CommitTree creates a commit from a given tree for the user with provided message.
func (r *SharedRepo) CommitTree(ctx context.Context, parent string, author, committer *rpc.Identity, treeHash,
	message string, signoff bool) (string, error) {
	return r.CommitTreeWithDate(ctx, parent, author, committer, treeHash, message, signoff, time.Now(), time.Now())
}

// CommitTreeWithDate creates a commit from a given tree for the user with provided message.
func (r *SharedRepo) CommitTreeWithDate(
	ctx context.Context,
	parent string,
	author, committer *rpc.Identity,
	treeHash, message string,
	signoff bool,
	authorDate, committerDate time.Time,
) (string, error) {
	committerSig := &git.Signature{
		Name:  committer.Name,
		Email: committer.Email,
		When:  time.Now(),
	}

	// Because this may call hooks we should pass in the environment
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME="+author.Name,
		"GIT_AUTHOR_EMAIL="+author.Email,
		"GIT_AUTHOR_DATE="+authorDate.Format(time.RFC3339),
		"GIT_COMMITTER_DATE="+committerDate.Format(time.RFC3339),
	)

	messageBytes := new(bytes.Buffer)
	_, _ = messageBytes.WriteString(message)
	_, _ = messageBytes.WriteString("\n")

	var args []string
	if parent != "" {
		args = []string{"commit-tree", treeHash, "-p", parent}
	} else {
		args = []string{"commit-tree", treeHash}
	}

	// temporary no signing
	args = append(args, "--no-gpg-sign")

	if signoff {
		// Signed-off-by
		_, _ = messageBytes.WriteString("\n")
		_, _ = messageBytes.WriteString("Signed-off-by: ")
		_, _ = messageBytes.WriteString(committerSig.String())
	}

	env = append(env,
		"GIT_COMMITTER_NAME="+committerSig.Name,
		"GIT_COMMITTER_EMAIL="+committerSig.Email,
	)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if err := git.NewCommand(ctx, args...).
		Run(&git.RunOpts{
			Env:    env,
			Dir:    r.basePath,
			Stdin:  messageBytes,
			Stdout: stdout,
			Stderr: stderr,
		}); err != nil {
		return "", fmt.Errorf("unable to commit-tree in temporary repo: %s Error: %w\nStdout: %s\nStderr: %s",
			r.repoUID, err, stdout, stderr)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Push the provided commitHash to the repository branch by the provided user.
func (r *SharedRepo) Push(ctx context.Context, doer *rpc.Identity, commitHash, branch string) error {
	// Because calls hooks we need to pass in the environment
	author, committer := doer, doer
	env := PushingEnvironment(author, committer, r.repoUID)
	if err := git.Push(ctx, r.basePath, git.PushOptions{
		Remote: r.remoteRepo.Path,
		Branch: strings.TrimSpace(commitHash) + ":" + git.BranchPrefix + strings.TrimSpace(branch),
		Env:    env,
	}); err != nil {
		if git.IsErrPushOutOfDate(err) {
			return err
		} else if git.IsErrPushRejected(err) {
			rejectErr := new(git.ErrPushRejected)
			if errors.As(err, &rejectErr) {
				log.Info().Msgf("Unable to push back to repo from temporary repo due to rejection:"+
					" %s (%s)\nStdout: %s\nStderr: %s\nError: %v",
					r.repoUID, r.basePath, rejectErr.StdOut, rejectErr.StdErr, rejectErr.Err)
			}
			return err
		}
		return fmt.Errorf("unable to push back to repo from temporary repo: %s (%s) Error: %w",
			r.repoUID, r.basePath, err)
	}
	return nil
}

// GetBranchCommit Gets the commit object of the given branch.
func (r *SharedRepo) GetBranchCommit(branch string) (*git.Commit, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("repository has not been cloned")
	}
	return r.repo.GetBranchCommit(branch)
}

// GetCommit Gets the commit object of the given commit ID.
func (r *SharedRepo) GetCommit(commitID string) (*git.Commit, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("repository has not been cloned")
	}
	return r.repo.GetCommit(commitID)
}

// PushingEnvironment returns an os environment to allow hooks to work on push.
func PushingEnvironment(
	author,
	committer *rpc.Identity,
	repoUID string,
) []string {
	authorSig := &git.Signature{
		Name:  author.Name,
		Email: author.Email,
	}
	committerSig := &git.Signature{
		Name:  committer.Name,
		Email: committer.Email,
	}

	environ := append(os.Environ(),
		"GIT_AUTHOR_NAME="+authorSig.Name,
		"GIT_AUTHOR_EMAIL="+authorSig.Email,
		"GIT_COMMITTER_NAME="+committerSig.Name,
		"GIT_COMMITTER_EMAIL="+committerSig.Email,
		// important env vars for hooks
		EnvPusherName+"="+committer.Name,
		// EnvPusherID+"="+fmt.Sprintf("%d", committer.ID),
		EnvRepoID+"="+repoUID,
		EnvAppURL+"=", // app url
	)

	return environ
}
