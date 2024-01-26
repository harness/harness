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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/tempdir"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
)

// SharedRepo is a type to wrap our upload repositories as a shallow clone.
type SharedRepo struct {
	adapter        Adapter
	repoUID        string
	repo           *gitea.Repository
	remoteRepoPath string
	tmpPath        string
}

// NewSharedRepo creates a new temporary upload repository.
func NewSharedRepo(
	adapter Adapter,
	baseTmpDir string,
	repoUID string,
	remoteRepoPath string,
) (*SharedRepo, error) {
	tmpPath, err := tempdir.CreateTemporaryPath(baseTmpDir, repoUID) // Need better solution
	if err != nil {
		return nil, err
	}

	t := &SharedRepo{
		adapter:        adapter,
		repoUID:        repoUID,
		remoteRepoPath: remoteRepoPath,
		tmpPath:        tmpPath,
	}
	return t, nil
}

func (r *SharedRepo) Path() string {
	return r.repo.Path
}

func (r *SharedRepo) RemotePath() string {
	return r.remoteRepoPath
}

// Close the repository cleaning up all files.
func (r *SharedRepo) Close(ctx context.Context) {
	defer r.repo.Close()
	if err := tempdir.RemoveTemporaryPath(r.tmpPath); err != nil {
		log.Ctx(ctx).Err(err).Msgf("Failed to remove temporary path %s", r.tmpPath)
	}
}

// filePriority is based on https://github.com/git/git/blob/master/tmp-objdir.c#L168
func filePriority(name string) int {
	switch {
	case !strings.HasPrefix(name, "pack"):
		return 0
	case strings.HasSuffix(name, ".keep"):
		return 1
	case strings.HasSuffix(name, ".pack"):
		return 2
	case strings.HasSuffix(name, ".rev"):
		return 3
	case strings.HasSuffix(name, ".idx"):
		return 4
	default:
		return 5
	}
}

type fileEntry struct {
	fileName string
	fullPath string
	relPath  string
	priority int
}

func (r *SharedRepo) MoveObjects(ctx context.Context) error {
	srcDir := path.Join(r.tmpPath, "objects")
	dstDir := path.Join(r.remoteRepoPath, "objects")

	var files []fileEntry

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// avoid coping anything in the info/
		if strings.HasPrefix(relPath, "info/") {
			return nil
		}

		fileName := filepath.Base(relPath)

		files = append(files, fileEntry{
			fileName: fileName,
			fullPath: path,
			relPath:  relPath,
			priority: filePriority(fileName),
		})

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk source directory: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].priority < files[j].priority // 0 is top priority, 5 is lowest priority
	})

	for _, f := range files {
		dstPath := filepath.Join(dstDir, f.relPath)

		err = os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
		if err != nil {
			return err
		}

		// Try to move the file

		errRename := os.Rename(f.fullPath, dstPath)
		if errRename == nil {
			log.Ctx(ctx).Debug().
				Str("object", f.relPath).
				Msg("moved git object")
			continue
		}

		// Try to copy the file

		copyError := func() error {
			srcFile, err := os.Open(f.fullPath)
			if err != nil {
				return fmt.Errorf("failed to open source file: %w", err)
			}
			defer func() { _ = srcFile.Close() }()

			dstFile, err := os.Create(dstPath)
			if err != nil {
				return fmt.Errorf("failed to create target file: %w", err)
			}
			defer func() { _ = dstFile.Close() }()

			_, err = io.Copy(dstFile, srcFile)
			if err != nil {
				return err
			}

			return nil
		}()
		if copyError != nil {
			log.Ctx(ctx).Err(copyError).
				Str("object", f.relPath).
				Str("renameErr", errRename.Error()).
				Msg("failed to move or copy git object")
			return copyError
		}

		log.Ctx(ctx).Warn().
			Str("object", f.relPath).
			Str("renameErr", errRename.Error()).
			Msg("copied git object")
	}

	return nil
}

func (r *SharedRepo) InitAsShared(ctx context.Context) error {
	args := []string{"init", "--bare"}
	if _, stderr, err := gitea.NewCommand(ctx, args...).RunStdString(&gitea.RunOpts{
		Dir: r.tmpPath,
	}); err != nil {
		return errors.Internal(err, "error while creating empty repository: %s", stderr)
	}

	if err := func() error {
		alternates := filepath.Join(r.tmpPath, "objects", "info", "alternates")
		f, err := os.OpenFile(alternates, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open alternates file '%s': %w", alternates, err)
		}
		defer func() { _ = f.Close() }()

		data := filepath.Join(r.remoteRepoPath, "objects")
		if _, err = fmt.Fprintln(f, data); err != nil {
			return fmt.Errorf("failed to write alternates file '%s': %w", alternates, err)
		}

		return nil
	}(); err != nil {
		return errors.Internal(err, "failed to create alternate in empty repository: %s", err.Error())
	}

	gitRepo, err := gitea.OpenRepository(ctx, r.tmpPath)
	if err != nil {
		return processGiteaErrorf(err, "failed to open repo")
	}

	r.repo = gitRepo

	return nil
}

// Clone the base repository to our path and set branch as the HEAD.
func (r *SharedRepo) Clone(ctx context.Context, branchName string) error {
	args := []string{"clone", "-s", "--bare"}
	if branchName != "" {
		args = append(args, "-b", strings.TrimPrefix(branchName, gitReferenceNamePrefixBranch))
	}
	args = append(args, r.remoteRepoPath, r.tmpPath)

	if _, _, err := gitea.NewCommand(ctx, args...).RunStdString(nil); err != nil {
		stderr := err.Error()
		if matched, _ := regexp.MatchString(".*Remote branch .* not found in upstream origin.*", stderr); matched {
			return errors.NotFound("branch '%s' does not exist", branchName)
		} else if matched, _ = regexp.MatchString(".* repository .* does not exist.*", stderr); matched {
			return errors.NotFound("repository '%s' does not exist", r.repoUID)
		}
		return errors.Internal(nil, "error while cloning repository: %s", stderr)
	}
	gitRepo, err := gitea.OpenRepository(ctx, r.tmpPath)
	if err != nil {
		return processGiteaErrorf(err, "failed to open repo")
	}
	r.repo = gitRepo
	return nil
}

// Init the repository.
func (r *SharedRepo) Init(ctx context.Context) error {
	if err := gitea.InitRepository(ctx, r.tmpPath, false); err != nil {
		return err
	}
	gitRepo, err := gitea.OpenRepository(ctx, r.tmpPath)
	if err != nil {
		return processGiteaErrorf(err, "failed to open repo")
	}
	r.repo = gitRepo
	return nil
}

// SetDefaultIndex sets the git index to our HEAD.
func (r *SharedRepo) SetDefaultIndex(ctx context.Context) error {
	if _, _, err := gitea.NewCommand(ctx, "read-tree", "HEAD").RunStdString(&gitea.RunOpts{Dir: r.tmpPath}); err != nil {
		return fmt.Errorf("failed to git read-tree HEAD: %w", err)
	}
	return nil
}

// SetIndex sets the git index to the provided treeish.
func (r *SharedRepo) SetIndex(ctx context.Context, treeish string) error {
	if _, _, err := gitea.NewCommand(ctx, "read-tree", treeish).RunStdString(&gitea.RunOpts{Dir: r.tmpPath}); err != nil {
		return fmt.Errorf("failed to git read-tree %s: %w", treeish, err)
	}
	return nil
}

// LsFiles checks if the given filename arguments are in the index.
func (r *SharedRepo) LsFiles(
	ctx context.Context,
	filenames ...string,
) ([]string, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)

	cmdArgs := []string{"ls-files", "-z", "--"}
	for _, arg := range filenames {
		if arg != "" {
			cmdArgs = append(cmdArgs, arg)
		}
	}

	if err := gitea.NewCommand(ctx, cmdArgs...).
		Run(&gitea.RunOpts{
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
func (r *SharedRepo) RemoveFilesFromIndex(
	ctx context.Context,
	filenames ...string,
) error {
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

	if err := gitea.NewCommand(ctx, "update-index", "--remove", "-z", "--index-info").
		Run(&gitea.RunOpts{
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
func (r *SharedRepo) WriteGitObject(
	ctx context.Context,
	content io.Reader,
) (string, error) {
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)

	if err := gitea.NewCommand(ctx, "hash-object", "-w", "--stdin").
		Run(&gitea.RunOpts{
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
func (r *SharedRepo) ShowFile(
	ctx context.Context,
	filePath string,
	commitHash string,
	writer io.Writer,
) error {
	stderr := new(bytes.Buffer)
	file := strings.TrimSpace(commitHash) + ":" + strings.TrimSpace(filePath)
	cmd := gitea.NewCommand(ctx, "show", file)
	if err := cmd.Run(&gitea.RunOpts{
		Dir:    r.repo.Path,
		Stdout: writer,
		Stderr: stderr,
	}); err != nil {
		return fmt.Errorf("show file: %w - %s", err, stderr)
	}
	return nil
}

// AddObjectToIndex adds the provided object hash to the index with the provided mode and path.
func (r *SharedRepo) AddObjectToIndex(
	ctx context.Context,
	mode string,
	objectHash string,
	objectPath string,
) error {
	if _, _, err := gitea.NewCommand(ctx, "update-index", "--add", "--replace", "--cacheinfo", mode, objectHash,
		objectPath).RunStdString(&gitea.RunOpts{Dir: r.tmpPath}); err != nil {
		if matched, _ := regexp.MatchString(".*Invalid path '.*", err.Error()); matched {
			return errors.InvalidArgument("invalid path '%s'", objectPath)
		}
		return fmt.Errorf("unable to add object to index at %s in temporary repo path %s Error: %w",
			objectPath, r.repoUID, err)
	}
	return nil
}

// WriteTree writes the current index as a tree to the object db and returns its hash.
func (r *SharedRepo) WriteTree(ctx context.Context) (string, error) {
	stdout, _, err := gitea.NewCommand(ctx, "write-tree").RunStdString(&gitea.RunOpts{Dir: r.tmpPath})
	if err != nil {
		return "", fmt.Errorf("unable to write-tree in temporary repo path for: %s Error: %w",
			r.repoUID, err)
	}
	return strings.TrimSpace(stdout), nil
}

// GetLastCommit gets the last commit ID SHA of the repo.
func (r *SharedRepo) GetLastCommit(ctx context.Context) (string, error) {
	return r.GetLastCommitByRef(ctx, "HEAD")
}

// GetLastCommitByRef gets the last commit ID SHA of the repo by ref.
func (r *SharedRepo) GetLastCommitByRef(
	ctx context.Context,
	ref string,
) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}
	stdout, _, err := gitea.NewCommand(ctx, "rev-parse", ref).RunStdString(&gitea.RunOpts{Dir: r.tmpPath})
	if err != nil {
		return "", processGiteaErrorf(err, "unable to rev-parse %s in temporary repo for: %s",
			ref, r.repoUID)
	}
	return strings.TrimSpace(stdout), nil
}

// CommitTreeWithDate creates a commit from a given tree for the user with provided message.
func (r *SharedRepo) CommitTreeWithDate(
	ctx context.Context,
	parent string,
	author, committer *types.Identity,
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
		giteaSignature := &gitea.Signature{
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
	if err := gitea.NewCommand(ctx, args...).
		Run(&gitea.RunOpts{
			Env:    env,
			Dir:    r.tmpPath,
			Stdin:  messageBytes,
			Stdout: stdout,
			Stderr: stderr,
		}); err != nil {
		return "", processGiteaErrorf(err, "unable to commit-tree in temporary repo: %s Error: %v\nStdout: %s\nStderr: %s",
			r.repoUID, err, stdout, stderr)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r *SharedRepo) PushDeleteBranch(
	ctx context.Context,
	branch string,
	force bool,
	env ...string,
) error {
	return r.push(ctx, "", GetReferenceFromBranchName(branch), force, env...)
}

func (r *SharedRepo) PushCommitToBranch(
	ctx context.Context,
	commitSHA string,
	branch string,
	force bool,
	env ...string,
) error {
	return r.push(ctx,
		commitSHA,
		GetReferenceFromBranchName(branch),
		force,
		env...,
	)
}

func (r *SharedRepo) PushBranch(
	ctx context.Context,
	sourceBranch string,
	branch string,
	force bool,
	env ...string,
) error {
	return r.push(ctx,
		GetReferenceFromBranchName(sourceBranch),
		GetReferenceFromBranchName(branch),
		force,
		env...,
	)
}
func (r *SharedRepo) PushTag(
	ctx context.Context,
	tagName string,
	force bool,
	env ...string,
) error {
	refTag := GetReferenceFromTagName(tagName)
	return r.push(ctx, refTag, refTag, force, env...)
}

func (r *SharedRepo) PushDeleteTag(
	ctx context.Context,
	tagName string,
	force bool,
	env ...string,
) error {
	refTag := GetReferenceFromTagName(tagName)
	return r.push(ctx, "", refTag, force, env...)
}

// push pushes the provided references to the provided branch in the original repository.
func (r *SharedRepo) push(
	ctx context.Context,
	sourceRef string,
	destinationRef string,
	force bool,
	env ...string,
) error {
	// Because calls hooks we need to pass in the environment
	if err := r.adapter.Push(ctx, r.tmpPath, types.PushOptions{
		Remote: r.remoteRepoPath,
		Branch: sourceRef + ":" + destinationRef,
		Env:    env,
		Force:  force,
	}); err != nil {
		return fmt.Errorf("unable to push back to repo from temporary repo: %w", err)
	}

	return nil
}

// GetBranchCommit Gets the commit object of the given branch.
func (r *SharedRepo) GetBranchCommit(branch string) (*gitea.Commit, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("repository has not been cloned")
	}

	return r.repo.GetBranchCommit(strings.TrimPrefix(branch, gitReferenceNamePrefixBranch))
}

// GetBranch gets the branch object of the given ref.
func (r *SharedRepo) GetBranch(rev string) (*gitea.Branch, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("repository has not been cloned")
	}
	return r.repo.GetBranch(rev)
}

// GetCommit Gets the commit object of the given commit ID.
func (r *SharedRepo) GetCommit(commitID string) (*gitea.Commit, error) {
	if r.repo == nil {
		return nil, fmt.Errorf("repository has not been cloned")
	}
	return r.repo.GetCommit(commitID)
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

// SharedRepository creates new instance of SharedRepo.
func (a Adapter) SharedRepository(
	tmpDir string,
	repoUID string,
	remotePath string,
) (*SharedRepo, error) {
	return NewSharedRepo(a, tmpDir, repoUID, remotePath)
}
