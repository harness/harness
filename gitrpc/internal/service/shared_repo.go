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

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc/internal/gitea"
	"github.com/harness/gitness/gitrpc/internal/middleware"
	"github.com/harness/gitness/gitrpc/internal/tempdir"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
)

// SharedRepo is a type to wrap our upload repositories as a shallow clone.
type SharedRepo struct {
	repoUID        string
	repo           *git.Repository
	remoteRepoPath string
	tmpPath        string
}

// NewSharedRepo creates a new temporary upload repository.
func NewSharedRepo(baseTmpDir, repoUID string, remoteRepoPath string) (*SharedRepo, error) {
	tmpPath, err := tempdir.CreateTemporaryPath(baseTmpDir, repoUID)
	if err != nil {
		return nil, err
	}
	t := &SharedRepo{
		repoUID:        repoUID,
		remoteRepoPath: remoteRepoPath,
		tmpPath:        tmpPath,
	}
	return t, nil
}

// Close the repository cleaning up all files.
func (r *SharedRepo) Close(ctx context.Context) {
	defer r.repo.Close()
	if err := tempdir.RemoveTemporaryPath(r.tmpPath); err != nil {
		log.Ctx(ctx).Err(err).Msgf("Failed to remove temporary path %s", r.tmpPath)
	}
}

// Clone the base repository to our path and set branch as the HEAD.
func (r *SharedRepo) Clone(ctx context.Context, branchName string) error {
	args := []string{"clone", "-s", "--bare"}
	if branchName != "" {
		args = append(args, "-b", strings.TrimPrefix(branchName, gitReferenceNamePrefixBranch))
	}
	args = append(args, r.remoteRepoPath, r.tmpPath)

	if _, _, err := git.NewCommand(ctx, args...).RunStdString(nil); err != nil {
		stderr := err.Error()
		if matched, _ := regexp.MatchString(".*Remote branch .* not found in upstream origin.*", stderr); matched {
			return git.ErrBranchNotExist{
				Name: branchName,
			}
		} else if matched, _ = regexp.MatchString(".* repository .* does not exist.*", stderr); matched {
			return fmt.Errorf("%s %w", r.repoUID, types.ErrNotFound)
		}
		return fmt.Errorf("Clone: %w %s", err, stderr)
	}
	gitRepo, err := git.OpenRepository(ctx, r.tmpPath)
	if err != nil {
		return processGitErrorf(err, "failed to open repo")
	}
	r.repo = gitRepo
	return nil
}

// Init the repository.
func (r *SharedRepo) Init(ctx context.Context) error {
	if err := git.InitRepository(ctx, r.tmpPath, false); err != nil {
		return err
	}
	gitRepo, err := git.OpenRepository(ctx, r.tmpPath)
	if err != nil {
		return processGitErrorf(err, "failed to open repo")
	}
	r.repo = gitRepo
	return nil
}

// SetDefaultIndex sets the git index to our HEAD.
func (r *SharedRepo) SetDefaultIndex(ctx context.Context) error {
	if _, _, err := git.NewCommand(ctx, "read-tree", "HEAD").RunStdString(&git.RunOpts{Dir: r.tmpPath}); err != nil {
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
			Dir:    r.tmpPath,
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
			Dir:    r.tmpPath,
			Stdin:  stdIn,
			Stdout: stdOut,
			Stderr: stdErr,
		}); err != nil {
		return fmt.Errorf("unable to update-index for temporary repo: %s Error: %w\nstdout: %s\nstderr: %s",
			r.repoUID, err, stdOut.String(), stdErr.String())
	}
	return nil
}

// WriteGitObject writes the provided content to the object db and returns its hash.
func (r *SharedRepo) WriteGitObject(ctx context.Context, content io.Reader) (string, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)

	if err := git.NewCommand(ctx, "hash-object", "-w", "--stdin").
		Run(&git.RunOpts{
			Dir:    r.tmpPath,
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
		objectPath).RunStdString(&git.RunOpts{Dir: r.tmpPath}); err != nil {
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
	stdout, _, err := git.NewCommand(ctx, "write-tree").RunStdString(&git.RunOpts{Dir: r.tmpPath})
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
	stdout, _, err := git.NewCommand(ctx, "rev-parse", ref).RunStdString(&git.RunOpts{Dir: r.tmpPath})
	if err != nil {
		return "", fmt.Errorf("unable to rev-parse %s in temporary repo for: %s Error: %w",
			ref, r.repoUID, err)
	}
	return strings.TrimSpace(stdout), nil
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
	// setup environment variables used by git-commit-tree
	// See https://git-scm.com/book/en/v2/Git-Internals-Environment-Variables
	env := []string{
		"GIT_AUTHOR_NAME=" + author.Name,
		"GIT_AUTHOR_EMAIL=" + author.Email,
		"GIT_AUTHOR_DATE=" + authorDate.Format(time.RFC3339),
		"GIT_COMMITTER_NAME=" + committer.Name,
		"GIT_COMMITTER_EMAIL=" + committer.Email,
		"GIT_COMMITTER_DATE=" + committerDate.Format(time.RFC3339),
	}
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
		giteaSignature := &git.Signature{
			Name:  committer.Name,
			Email: committer.Email,
			When:  committerDate,
		}
		// Signed-off-by
		_, _ = messageBytes.WriteString("\n")
		_, _ = messageBytes.WriteString("Signed-off-by: ")
		_, _ = messageBytes.WriteString(giteaSignature.String())
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if err := git.NewCommand(ctx, args...).
		Run(&git.RunOpts{
			Env:    env,
			Dir:    r.tmpPath,
			Stdin:  messageBytes,
			Stdout: stdout,
			Stderr: stderr,
		}); err != nil {
		return "", fmt.Errorf("unable to commit-tree in temporary repo: %s Error: %w\nStdout: %s\nStderr: %s",
			r.repoUID, err, stdout, stderr)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r *SharedRepo) PushDeleteBranch(ctx context.Context, writeRequest *rpc.WriteRequest,
	branch string) error {
	return r.push(ctx, writeRequest, "", GetReferenceFromBranchName(branch))
}

func (r *SharedRepo) PushCommitToBranch(ctx context.Context, writeRequest *rpc.WriteRequest,
	commitSHA string, branch string) error {
	return r.push(ctx, writeRequest, commitSHA, GetReferenceFromBranchName(branch))
}

func (r *SharedRepo) PushBranch(ctx context.Context, writeRequest *rpc.WriteRequest,
	sourceBranch string, branch string) error {
	return r.push(ctx, writeRequest, GetReferenceFromBranchName(sourceBranch), GetReferenceFromBranchName(branch))
}
func (r *SharedRepo) PushTag(ctx context.Context, writeRequest *rpc.WriteRequest,
	tagName string) error {
	refTag := GetReferenceFromTagName(tagName)
	return r.push(ctx, writeRequest, refTag, refTag)
}

func (r *SharedRepo) PushDeleteTag(ctx context.Context, writeRequest *rpc.WriteRequest,
	tagName string) error {
	refTag := GetReferenceFromTagName(tagName)
	return r.push(ctx, writeRequest, "", refTag)
}

// push pushes the provided references to the provided branch in the original repository.
func (r *SharedRepo) push(ctx context.Context, writeRequest *rpc.WriteRequest,
	sourceRef, destinationRef string) error {
	// Because calls hooks we need to pass in the environment
	env := CreateEnvironmentForPush(ctx, writeRequest)
	if err := gitea.Push(ctx, r.tmpPath, types.PushOptions{
		Remote: r.remoteRepoPath,
		Branch: sourceRef + ":" + destinationRef,
		Env:    env,
	}); err != nil {
		if git.IsErrPushOutOfDate(err) {
			return err
		} else if git.IsErrPushRejected(err) {
			rejectErr := new(git.ErrPushRejected)
			if errors.As(err, &rejectErr) {
				log.Ctx(ctx).Info().Msgf("Unable to push back to repo from temporary repo due to rejection:"+
					" %s (%s)\nStdout: %s\nStderr: %s\nError: %v",
					r.repoUID, r.tmpPath, rejectErr.StdOut, rejectErr.StdErr, rejectErr.Err)
			}
			return err
		}
		return fmt.Errorf("unable to push back to repo from temporary repo: %s (%s) Error: %w",
			r.repoUID, r.tmpPath, err)
	}
	return nil
}

// GetBranchCommit Gets the commit object of the given branch.
func (r *SharedRepo) GetBranchCommit(branch string) (*git.Commit, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("repository has not been cloned")
	}

	return r.repo.GetBranchCommit(strings.TrimPrefix(branch, gitReferenceNamePrefixBranch))
}

// GetCommit Gets the commit object of the given commit ID.
func (r *SharedRepo) GetCommit(commitID string) (*git.Commit, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("repository has not been cloned")
	}
	return r.repo.GetCommit(commitID)
}

// ASSUMPTION: writeRequst and writeRequst.Actor is never nil.
func CreateEnvironmentForPush(ctx context.Context, writeRequest *rpc.WriteRequest) []string {
	// don't send existing environment variables (os.Environ()), only send what's explicitly necessary.
	// Otherwise we create implicit dependencies that are easy to break.
	environ := []string{
		// request id to use for hooks
		EnvRequestID + "=" + middleware.RequestIDFrom(ctx),
		// repo related info
		EnvRepoUID + "=" + writeRequest.RepoUid,
		// actor related info
		EnvActorName + "=" + writeRequest.Actor.Name,
		EnvActorEmail + "=" + writeRequest.Actor.Email,
	}

	// add all environment variables coming from client request
	for _, envVar := range writeRequest.EnvVars {
		environ = append(environ, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
	}

	// add all environment variables from the metadata
	if metadata, mOK := metadata.FromIncomingContext(ctx); mOK {
		if envVars, eOK := metadata[rpc.MetadataKeyEnvironmentVariables]; eOK {
			// TODO: should we do a sanity check?
			environ = append(environ, envVars...)
		}
	}

	return environ
}

// GetReferenceFromBranchName assumes the provided value is the branch name (not the ref!)
// and first sanitizes the branch name (remove any spaces or 'refs/heads/' prefix)
// It then returns the full form of the branch reference.
func GetReferenceFromBranchName(branchName string) string {
	// remove spaces
	branchName = strings.TrimSpace(branchName)
	// remove `refs/heads/` prefix (shouldn't be there, but if it is remove it to try to avoid complications)
	// NOTE: This is used to reduce missconfigurations via api
	// TODO: block via CLI, too
	branchName = strings.TrimPrefix(branchName, gitReferenceNamePrefixBranch)

	// return reference
	return gitReferenceNamePrefixBranch + branchName
}

func GetReferenceFromTagName(tagName string) string {
	// remove spaces
	tagName = strings.TrimSpace(tagName)
	// remove `refs/heads/` prefix (shouldn't be there, but if it is remove it to try to avoid complications)
	// NOTE: This is used to reduce missconfigurations via api
	// TODO: block via CLI, too
	tagName = strings.TrimPrefix(tagName, gitReferenceNamePrefixTag)

	// return reference
	return gitReferenceNamePrefixTag + tagName
}
