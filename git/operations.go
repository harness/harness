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

package git

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/adapter"
	"github.com/harness/gitness/git/types"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/services/repository/files"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

const (
	filePrefix            = "file://"
	defaultFilePermission = "100644" // 0o644 default file permission
)

type FileAction string

const (
	CreateAction FileAction = "CREATE"
	UpdateAction FileAction = "UPDATE"
	DeleteAction            = "DELETE"
	MoveAction              = "MOVE"
)

func (FileAction) Enum() []interface{} {
	return []interface{}{CreateAction, UpdateAction, DeleteAction, MoveAction}
}

// CommitFileAction holds file operation data.
type CommitFileAction struct {
	Action  FileAction
	Path    string
	Payload []byte
	SHA     string
}

// CommitFilesParams holds the data for file operations.
type CommitFilesParams struct {
	WriteParams
	Title     string
	Message   string
	Branch    string
	NewBranch string
	Actions   []CommitFileAction

	// Committer overwrites the git committer used for committing the files
	// (optional, default: actor)
	Committer *Identity
	// CommitterDate overwrites the git committer date used for committing the files
	// (optional, default: current time on server)
	CommitterDate *time.Time
	// Author overwrites the git author used for committing the files
	// (optional, default: committer)
	Author *Identity
	// AuthorDate overwrites the git author date used for committing the files
	// (optional, default: committer date)
	AuthorDate *time.Time
}

func (p *CommitFilesParams) Validate() error {
	return p.WriteParams.Validate()
}

type CommitFilesResponse struct {
	CommitID string
}

func (s *Service) CommitFiles(ctx context.Context, params *CommitFilesParams) (CommitFilesResponse, error) {
	if err := params.Validate(); err != nil {
		return CommitFilesResponse{}, err
	}

	log := log.Ctx(ctx).With().Str("repo_uid", params.RepoUID).Logger()

	committer := params.Actor
	if params.Committer != nil {
		committer = *params.Committer
	}
	committerDate := time.Now().UTC()
	if params.CommitterDate != nil {
		committerDate = *params.CommitterDate
	}

	author := committer
	if params.Author != nil {
		author = *params.Author
	}
	authorDate := committerDate
	if params.AuthorDate != nil {
		authorDate = *params.AuthorDate
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	log.Debug().Msg("open repository")

	repo, err := s.adapter.OpenRepository(ctx, repoPath)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("CommitFiles: failed to open repo: %w", err)
	}

	log.Debug().Msg("check if empty")

	// check if repo is empty
	// IMPORTANT: we don't use gitea's repo.IsEmpty() as that only checks whether the default branch exists (in HEAD).
	// This can be an issue in case someone created a branch already in the repo (just default branch is missing).
	// In that case the user can accidentally create separate git histories (which most likely is unintended).
	// If the user wants to actually build a disconnected commit graph they can use the cli.
	isEmpty, err := s.adapter.HasBranches(ctx, repoPath)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("CommitFiles: failed to determine if repository is empty: %w", err)
	}

	log.Debug().Msg("validate and prepare input")

	// ensure input data is valid
	if err = s.validateAndPrepareHeader(repo, isEmpty, params); err != nil {
		return CommitFilesResponse{}, err
	}

	log.Debug().Msg("create shared repo")

	// create a new shared repo
	shared, err := s.adapter.SharedRepository(s.tmpDir, params.RepoUID, repo.Path)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to create shared repository: %w", err)
	}
	defer shared.Close(ctx)

	log.Debug().Msgf("prepare tree (empty: %t)", isEmpty)

	// handle empty repo separately (as branch doesn't exist, no commit exists, ...)
	var parentCommitSHA string
	if isEmpty {
		err = s.prepareTreeEmptyRepo(ctx, shared, params.Actions)
		if err != nil {
			return CommitFilesResponse{}, err
		}
	} else {
		parentCommitSHA, err = s.prepareTree(ctx, shared, params.Branch, params.Actions)
		if err != nil {
			return CommitFilesResponse{}, err
		}
	}

	log.Debug().Msg("write tree")

	// Now write the tree
	treeHash, err := shared.WriteTree(ctx)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to write tree object: %w", err)
	}

	message := strings.TrimSpace(params.Title)
	if len(params.Message) > 0 {
		message += "\n\n" + strings.TrimSpace(params.Message)
	}

	log.Debug().Msg("commit tree")

	// Now commit the tree
	commitSHA, err := shared.CommitTreeWithDate(
		ctx,
		parentCommitSHA,
		&types.Identity{
			Name:  author.Name,
			Email: author.Email,
		},
		&types.Identity{
			Name:  committer.Name,
			Email: committer.Email,
		},
		treeHash,
		message,
		false,
		authorDate,
		committerDate,
	)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to commit the tree: %w", err)
	}

	log.Debug().Msg("push branch to original repo")

	env := CreateEnvironmentForPush(ctx, params.WriteParams)
	if err = shared.PushCommitToBranch(ctx, commitSHA, params.NewBranch, false, env...); err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to push commits to remote repository: %w", err)
	}

	log.Debug().Msg("get commit")

	commit, err := shared.GetCommit(commitSHA)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to get commit for SHA %s: %w", commitSHA, err)
	}

	log.Debug().Msg("done")

	return CommitFilesResponse{
		CommitID: commit.ID.String(),
	}, nil
}

func (s *Service) prepareTree(ctx context.Context, shared SharedRepo,
	branchName string, actions []CommitFileAction) (string, error) {
	// clone original branch from repo
	if err := s.clone(ctx, shared, branchName); err != nil {
		return "", err
	}

	// Get the latest commit of the original branch
	commit, err := shared.GetBranchCommit(branchName)
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit of the branch %s: %w", branchName, err)
	}

	// execute all actions
	for _, action := range actions {
		action := action
		if err = s.processAction(ctx, shared, &action, commit); err != nil {
			return "", err
		}
	}

	return commit.ID.String(), nil
}

func (s *Service) prepareTreeEmptyRepo(ctx context.Context, shared *adapter.SharedRepo,
	actions []CommitFileAction) error {
	// init a new repo (full clone would cause risk that by time of push someone wrote to the remote repo!)
	err := shared.Init(ctx)
	if err != nil {
		return fmt.Errorf("failed to init shared tmp repository: %w", err)
	}

	for _, action := range actions {
		if action.Action != CreateAction {
			return errors.PreconditionFailed("action not allowed on empty repository")
		}

		filePath := files.CleanUploadFileName(action.Path)
		if filePath == "" {
			return errors.InvalidArgument("invalid path")
		}

		reader := bytes.NewReader(action.Payload)
		if err = createFile(ctx, shared, nil, filePath, defaultFilePermission, reader); err != nil {
			return errors.Internal("failed to create file '%s': %w", action.Path, err)
		}
	}

	return nil
}

func (s *Service) validateAndPrepareHeader(repo *git.Repository, isEmpty bool,
	params *CommitFilesParams) error {
	if params.Branch == "" {
		defaultBranchRef, err := repo.GetDefaultBranch()
		if err != nil {
			return fmt.Errorf("failed to get default branch: %w", err)
		}
		params.Branch = defaultBranchRef
	}

	if params.NewBranch == "" {
		params.NewBranch = params.Branch
	}

	// trim refs/heads/ prefixes to avoid issues when calling gitea API
	params.Branch = strings.TrimPrefix(strings.TrimSpace(params.Branch), gitReferenceNamePrefixBranch)
	params.NewBranch = strings.TrimPrefix(strings.TrimSpace(params.NewBranch), gitReferenceNamePrefixBranch)

	// if the repo is empty then we can skip branch existence checks
	if isEmpty {
		return nil
	}

	// ensure source branch exists
	if _, err := repo.GetBranch(params.Branch); err != nil {
		return fmt.Errorf("failed to get source branch '%s': %w", params.Branch, err)
	}

	// ensure new branch doesn't exist yet (if new branch creation was requested)
	if params.Branch != params.NewBranch {
		existingBranch, err := repo.GetBranch(params.NewBranch)
		if existingBranch != nil {
			return errors.Conflict("branch %s already exists", existingBranch.Name)
		}
		if err != nil && !git.IsErrBranchNotExist(err) {
			return fmt.Errorf("failed to create new branch '%s': %w", params.NewBranch, err)
		}
	}
	return nil
}

func (s *Service) clone(
	ctx context.Context,
	shared SharedRepo,
	branch string,
) error {
	if err := shared.Clone(ctx, branch); err != nil {
		return fmt.Errorf("failed to clone branch '%s': %w", branch, err)
	}

	if err := shared.SetDefaultIndex(ctx); err != nil {
		return fmt.Errorf("failed to set default index: %w", err)
	}

	return nil
}

func (s *Service) processAction(
	ctx context.Context,
	shared SharedRepo,
	action *CommitFileAction,
	commit *git.Commit,
) (err error) {
	filePath := files.CleanUploadFileName(action.Path)
	if filePath == "" {
		return errors.InvalidArgument("invalid path: %w", err)
	}

	reader := bytes.NewReader(action.Payload)

	switch action.Action {
	case CreateAction:
		err = createFile(ctx, shared, commit, filePath, defaultFilePermission, reader)
	case UpdateAction:
		err = updateFile(ctx, shared, commit, filePath, action.SHA, defaultFilePermission, reader)
	case MoveAction:
		err = moveFile(ctx, shared, commit, filePath, defaultFilePermission, reader)
	case DeleteAction:
		err = deleteFile(ctx, shared, filePath)
	}

	return err
}

func createFile(ctx context.Context, repo SharedRepo, commit *git.Commit,
	filePath, mode string, reader io.Reader) error {
	// only check path availability if a source commit is available (empty repo won't have such a commit)
	if commit != nil {
		if err := checkPathAvailability(commit, filePath, true); err != nil {
			return err
		}
	}

	hash, err := repo.WriteGitObject(ctx, reader)
	if err != nil {
		return fmt.Errorf("createFile: error hashing object: %w", err)
	}

	// Add the object to the index
	if err = repo.AddObjectToIndex(ctx, mode, hash, filePath); err != nil {
		return fmt.Errorf("createFile: error creating object: %w", err)
	}
	return nil
}

func updateFile(ctx context.Context, repo SharedRepo, commit *git.Commit, filePath, sha,
	mode string, reader io.Reader) error {
	// get file mode from existing file (default unless executable)
	entry, err := getFileEntry(commit, sha, filePath)
	if err != nil {
		return err
	}
	if entry.IsExecutable() {
		mode = "100755"
	}

	hash, err := repo.WriteGitObject(ctx, reader)
	if err != nil {
		return fmt.Errorf("updateFile: error hashing object: %w", err)
	}

	if err = repo.AddObjectToIndex(ctx, mode, hash, filePath); err != nil {
		return fmt.Errorf("updateFile: error updating object: %w", err)
	}
	return nil
}

func moveFile(ctx context.Context, repo SharedRepo, commit *git.Commit,
	filePath, mode string, reader io.Reader) error {
	buffer := &bytes.Buffer{}
	newPath, err := parsePayload(reader, buffer)
	if err != nil {
		return err
	}

	if buffer.Len() == 0 && newPath != "" {
		err = repo.ShowFile(ctx, filePath, commit.ID.String(), buffer)
		if err != nil {
			return fmt.Errorf("moveFile: failed lookup for path '%s': %w", newPath, err)
		}
	}

	if err = checkPathAvailability(commit, newPath, false); err != nil {
		return err
	}

	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return fmt.Errorf("moveFile: listing files error: %w", err)
	}
	if !slices.Contains(filesInIndex, filePath) {
		return errors.NotFound("path %s not found", filePath)
	}

	hash, err := repo.WriteGitObject(ctx, buffer)
	if err != nil {
		return fmt.Errorf("moveFile: error hashing object: %w", err)
	}

	if err = repo.AddObjectToIndex(ctx, mode, hash, newPath); err != nil {
		return fmt.Errorf("moveFile: add object error: %w", err)
	}

	if err = repo.RemoveFilesFromIndex(ctx, filePath); err != nil {
		return fmt.Errorf("moveFile: remove object error: %w", err)
	}
	return nil
}

func deleteFile(ctx context.Context, repo SharedRepo, filePath string) error {
	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return fmt.Errorf("deleteFile: listing files error: %w", err)
	}
	if !slices.Contains(filesInIndex, filePath) {
		return errors.NotFound("file path %s not found", filePath)
	}

	if err = repo.RemoveFilesFromIndex(ctx, filePath); err != nil {
		return fmt.Errorf("deleteFile: remove object error: %w", err)
	}
	return nil
}

func getFileEntry(
	commit *git.Commit,
	sha string,
	path string,
) (*git.TreeEntry, error) {
	entry, err := commit.GetTreeEntryByPath(path)
	if git.IsErrNotExist(err) {
		return nil, errors.NotFound("path %s not found", path)
	}
	if err != nil {
		return nil, fmt.Errorf("getFileEntry: failed to get tree for path %s: %w", path, err)
	}

	// If a SHA was given and the SHA given doesn't match the SHA of the fromTreePath, throw error
	if sha == "" || sha != entry.ID.String() {
		return nil, errors.InvalidArgument("sha does not match for path %s [given: %s, expected: %s]",
			path, sha, entry.ID.String())
	}

	return entry, nil
}

// checkPathAvailability ensures that the path is available for the requested operation.
// For the path where this file will be created/updated, we need to make
// sure no parts of the path are existing files or links except for the last
// item in the path which is the file name, and that shouldn't exist IF it is
// a new file OR is being moved to a new path.
func checkPathAvailability(commit *git.Commit, filePath string, isNewFile bool) error {
	parts := strings.Split(filePath, "/")
	subTreePath := ""
	for index, part := range parts {
		subTreePath = path.Join(subTreePath, part)
		entry, err := commit.GetTreeEntryByPath(subTreePath)
		if err != nil {
			if git.IsErrNotExist(err) {
				// Means there is no item with that name, so we're good
				break
			}
			return fmt.Errorf("checkPathAvailability: failed to get tree entry for path %s: %w", subTreePath, err)
		}
		switch {
		case index < len(parts)-1:
			if !entry.IsDir() {
				return errors.Conflict("a file already exists where you're trying to create a subdirectory [path: %s]",
					subTreePath)
			}
		case entry.IsLink():
			return fmt.Errorf("a symbolic link %w where you're trying to create a subdirectory [path: %s]",
				types.ErrAlreadyExists, subTreePath)
		case entry.IsDir():
			return errors.Conflict("a directory already exists where you're trying to create a subdirectory [path: %s]",
				subTreePath)
		case filePath != "" || isNewFile:
			return errors.Conflict("file path %s already exists", filePath)
		}
	}
	return nil
}

func parsePayload(payload io.Reader, content io.Writer) (string, error) {
	newPath := ""
	reader := bufio.NewReader(payload)
	// check for filePrefix
	prefixBytes := make([]byte, len(filePrefix))
	if _, err := reader.Read(prefixBytes); err != nil {
		if errors.Is(err, io.EOF) {
			return "", nil
		}
		return "", err
	}
	// check if payload starts with filePrefix constant
	if bytes.Equal(prefixBytes, []byte(filePrefix)) {
		filename, _ := reader.ReadString('\n') // no err handling because next statement will check filename
		newPath = files.CleanUploadFileName(filename)
		if newPath == "" {
			return "", types.ErrInvalidPath
		}
	} else {
		if _, err := content.Write(prefixBytes); err != nil {
			return "", err
		}
	}
	_, err := io.Copy(content, reader)
	return newPath, err
}
