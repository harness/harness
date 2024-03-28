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
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/tempdir"

	"github.com/rs/zerolog/log"
)

// SharedRepo is a type to wrap our upload repositories as a shallow clone.
type SharedRepo struct {
	git            *Git
	repoUID        string
	remoteRepoPath string
	RepoPath       string
}

// NewSharedRepo creates a new temporary upload repository.
func NewSharedRepo(
	adapter *Git,
	baseTmpDir string,
	repoUID string,
	remoteRepoPath string,
) (*SharedRepo, error) {
	tmpPath, err := tempdir.CreateTemporaryPath(baseTmpDir, repoUID) // Need better solution
	if err != nil {
		return nil, err
	}

	t := &SharedRepo{
		git:            adapter,
		repoUID:        repoUID,
		remoteRepoPath: remoteRepoPath,
		RepoPath:       tmpPath,
	}
	return t, nil
}

// Close the repository cleaning up all files.
func (r *SharedRepo) Close(ctx context.Context) {
	if err := tempdir.RemoveTemporaryPath(r.RepoPath); err != nil {
		log.Ctx(ctx).Err(err).Msgf("Failed to remove temporary path %s", r.RepoPath)
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
	srcDir := path.Join(r.RepoPath, "objects")
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
func (r *SharedRepo) initRepository(ctx context.Context, alternateObjDirs ...string) error {
	cmd := command.New("init", command.WithFlag("--bare"))

	if err := cmd.Run(ctx, command.WithDir(r.RepoPath)); err != nil {
		return errors.Internal(err, "error while creating empty repository")
	}

	if err := func() error {
		alternates := filepath.Join(r.RepoPath, "objects", "info", "alternates")
		f, err := os.OpenFile(alternates, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open alternates file '%s': %w", alternates, err)
		}
		defer func() { _ = f.Close() }()

		data := strings.Join(
			append(
				alternateObjDirs,
				filepath.Join(r.remoteRepoPath, "objects"),
			),
			"\n",
		)

		if _, err = fmt.Fprintln(f, data); err != nil {
			return fmt.Errorf("failed to write alternates file '%s': %w", alternates, err)
		}

		return nil
	}(); err != nil {
		return errors.Internal(err, "failed to create alternate in empty repository: %s", err.Error())
	}

	return nil
}

func (r *SharedRepo) InitAsShared(ctx context.Context) error {
	return r.initRepository(ctx)
}

// InitAsSharedWithAlternates initializes repository with provided alternate object directories.
func (r *SharedRepo) InitAsSharedWithAlternates(ctx context.Context, alternateObjDirs ...string) error {
	return r.initRepository(ctx, alternateObjDirs...)
}

// Clone the base repository to our path and set branch as the HEAD.
func (r *SharedRepo) Clone(ctx context.Context, branchName string) error {
	cmd := command.New("clone",
		command.WithFlag("-s"),
		command.WithFlag("--bare"),
	)
	if branchName != "" {
		cmd.Add(command.WithFlag("-b", strings.TrimPrefix(branchName, gitReferenceNamePrefixBranch)))
	}
	cmd.Add(command.WithArg(r.remoteRepoPath, r.RepoPath))

	if err := cmd.Run(ctx); err != nil {
		cmderr := command.AsError(err)
		if cmderr.StdErr == nil {
			return errors.Internal(err, "error while cloning repository")
		}
		stderr := string(cmderr.StdErr)
		matched, _ := regexp.MatchString(".*Remote branch .* not found in upstream origin.*", stderr)
		if matched {
			return errors.NotFound("branch '%s' does not exist", branchName)
		}
		matched, _ = regexp.MatchString(".* repository .* does not exist.*", stderr)
		if matched {
			return errors.NotFound("repository '%s' does not exist", r.repoUID)
		}
	}
	return nil
}

// Init the repository.
func (r *SharedRepo) Init(ctx context.Context) error {
	err := r.git.InitRepository(ctx, r.RepoPath, false)
	if err != nil {
		return fmt.Errorf("failed to initialize shared repo: %w", err)
	}
	return nil
}

// SetDefaultIndex sets the git index to our HEAD.
func (r *SharedRepo) SetDefaultIndex(ctx context.Context) error {
	return r.SetIndex(ctx, "HEAD")
}

// SetIndex sets the git index to the provided treeish.
func (r *SharedRepo) SetIndex(ctx context.Context, rev string) error {
	cmd := command.New("read-tree", command.WithArg(rev))
	if err := cmd.Run(ctx, command.WithDir(r.RepoPath)); err != nil {
		return fmt.Errorf("failed to git read-tree %s: %w", rev, err)
	}
	return nil
}

// LsFiles checks if the given filename arguments are in the index.
func (r *SharedRepo) LsFiles(
	ctx context.Context,
	filenames ...string,
) ([]string, error) {
	cmd := command.New("ls-files",
		command.WithFlag("-z"),
	)

	for _, arg := range filenames {
		if arg != "" {
			cmd.Add(command.WithPostSepArg(arg))
		}
	}

	stdout := bytes.NewBuffer(nil)

	err := cmd.Run(ctx,
		command.WithDir(r.RepoPath),
		command.WithStdout(stdout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in shared repository's git index: %w", err)
	}

	files := make([]string, 0)
	for _, line := range bytes.Split(stdout.Bytes(), []byte{'\000'}) {
		files = append(files, string(line))
	}

	return files, nil
}

// RemoveFilesFromIndex removes the given files from the index.
func (r *SharedRepo) RemoveFilesFromIndex(
	ctx context.Context,
	filenames ...string,
) error {
	stdOut := new(bytes.Buffer)
	stdIn := new(bytes.Buffer)
	for _, file := range filenames {
		if file != "" {
			stdIn.WriteString("0 0000000000000000000000000000000000000000\t")
			stdIn.WriteString(file)
			stdIn.WriteByte('\000')
		}
	}

	cmd := command.New("update-index",
		command.WithFlag("--remove"),
		command.WithFlag("-z"),
		command.WithFlag("--index-info"),
	)

	if err := cmd.Run(ctx,
		command.WithDir(r.RepoPath),
		command.WithStdin(stdIn),
		command.WithStdout(stdOut),
	); err != nil {
		return fmt.Errorf("unable to update-index for temporary repo: %s Error: %w\nstdout: %s",
			r.repoUID, err, stdOut.String())
	}
	return nil
}

// WriteGitObject writes the provided content to the object db and returns its hash.
func (r *SharedRepo) WriteGitObject(
	ctx context.Context,
	content io.Reader,
) (sha.SHA, error) {
	stdOut := new(bytes.Buffer)
	cmd := command.New("hash-object",
		command.WithFlag("-w"),
		command.WithFlag("--stdin"),
	)
	if err := cmd.Run(ctx,
		command.WithDir(r.RepoPath),
		command.WithStdin(content),
		command.WithStdout(stdOut),
	); err != nil {
		return sha.None, fmt.Errorf("unable to hash-object to temporary repo: %s Error: %w\nstdout: %s",
			r.repoUID, err, stdOut.String())
	}

	return sha.New(stdOut.String())
}

// ShowFile dumps show file and write to io.Writer.
func (r *SharedRepo) ShowFile(
	ctx context.Context,
	filePath string,
	commitHash string,
	writer io.Writer,
) error {
	file := strings.TrimSpace(commitHash) + ":" + strings.TrimSpace(filePath)
	cmd := command.New("show",
		command.WithArg(file),
	)
	if err := cmd.Run(ctx,
		command.WithDir(r.RepoPath),
		command.WithStdout(writer),
	); err != nil {
		return fmt.Errorf("show file: %w", err)
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
	cmd := command.New("update-index",
		command.WithFlag("--add"),
		command.WithFlag("--replace"),
		command.WithFlag("--cacheinfo"),
		command.WithArg(mode, objectHash, objectPath),
	)
	if err := cmd.Run(ctx, command.WithDir(r.RepoPath)); err != nil {
		if matched, _ := regexp.MatchString(".*Invalid path '.*", err.Error()); matched {
			return errors.InvalidArgument("invalid path '%s'", objectPath)
		}
		return fmt.Errorf("unable to add object to index at %s in temporary repo path %s Error: %w",
			objectPath, r.repoUID, err)
	}
	return nil
}

// WriteTree writes the current index as a tree to the object db and returns its hash.
func (r *SharedRepo) WriteTree(ctx context.Context) (sha.SHA, error) {
	stdout := &bytes.Buffer{}
	cmd := command.New("write-tree")
	err := cmd.Run(ctx,
		command.WithDir(r.RepoPath),
		command.WithStdout(stdout),
	)
	if err != nil {
		return sha.None, fmt.Errorf("unable to write-tree in temporary repo path for: %s Error: %w",
			r.repoUID, err)
	}
	return sha.New(stdout.String())
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
	stdout := &bytes.Buffer{}
	cmd := command.New("rev-parse",
		command.WithArg(ref),
	)
	err := cmd.Run(ctx,
		command.WithDir(r.RepoPath),
		command.WithStdout(stdout),
	)
	if err != nil {
		return "", processGitErrorf(err, "unable to rev-parse %s in temporary repo for: %s",
			ref, r.repoUID)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// CommitTreeWithDate creates a commit from a given tree for the user with provided message.
func (r *SharedRepo) CommitTreeWithDate(
	ctx context.Context,
	parent sha.SHA,
	author, committer *Identity,
	treeHash sha.SHA,
	message string,
	signoff bool,
	authorDate, committerDate time.Time,
) (sha.SHA, error) {
	messageBytes := new(bytes.Buffer)
	_, _ = messageBytes.WriteString(message)
	_, _ = messageBytes.WriteString("\n")

	cmd := command.New("commit-tree",
		command.WithAuthorAndDate(
			author.Name,
			author.Email,
			authorDate,
		),
		command.WithCommitterAndDate(
			committer.Name,
			committer.Email,
			committerDate,
		),
	)
	if !parent.IsEmpty() {
		cmd.Add(command.WithFlag("-p", parent.String()))
	}
	cmd.Add(command.WithArg(treeHash.String()))

	// temporary no signing
	cmd.Add(command.WithFlag("--no-gpg-sign"))

	if signoff {
		sig := &Signature{
			Identity: Identity{
				Name:  committer.Name,
				Email: committer.Email,
			},
			When: committerDate,
		}
		// Signed-off-by
		_, _ = messageBytes.WriteString("\n")
		_, _ = messageBytes.WriteString("Signed-off-by: ")
		_, _ = messageBytes.WriteString(sig.String())
	}

	stdout := new(bytes.Buffer)
	if err := cmd.Run(ctx,
		command.WithDir(r.RepoPath),
		command.WithStdin(messageBytes),
		command.WithStdout(stdout),
	); err != nil {
		return sha.None, processGitErrorf(err, "unable to commit-tree in temporary repo: %s Error: %v\nStdout: %s",
			r.repoUID, err, stdout)
	}
	return sha.New(stdout.String())
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
	if err := r.git.Push(ctx, r.RepoPath, PushOptions{
		Remote: r.remoteRepoPath,
		Branch: sourceRef + ":" + destinationRef,
		Env:    env,
		Force:  force,
	}); err != nil {
		return fmt.Errorf("unable to push back to repo from temporary repo: %w", err)
	}

	return nil
}

// GetBranch gets the branch object of the given ref.
func (r *SharedRepo) GetBranch(ctx context.Context, rev string) (*Branch, error) {
	return r.git.GetBranch(ctx, r.RepoPath, rev)
}

// GetCommit Gets the commit object of the given commit ID.
func (r *SharedRepo) GetCommit(ctx context.Context, commitID string) (*Commit, error) {
	return r.git.GetCommit(ctx, r.RepoPath, commitID)
}

// GetTreeNode Gets the tree node object of the given commit ID and path.
func (r *SharedRepo) GetTreeNode(ctx context.Context, commitID, treePath string) (*TreeNode, error) {
	return r.git.GetTreeNode(ctx, r.RepoPath, commitID, treePath)
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
