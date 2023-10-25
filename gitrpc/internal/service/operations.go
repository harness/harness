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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc/internal/files"
	"github.com/harness/gitness/gitrpc/internal/slices"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
)

const (
	filePrefix            = "file://"
	defaultFilePermission = "100644" // 0o644 default file permission
)

type CommitFilesService struct {
	rpc.UnimplementedCommitFilesServiceServer
	adapter      GitAdapter
	reposRoot    string
	reposTempDir string
}

type fileAction struct {
	header *rpc.CommitFilesActionHeader
	// content can hold file content or new path for move operation
	// new path is prefixed with filePrefix constant
	content []byte
}

func NewCommitFilesService(adapter GitAdapter, reposRoot, reposTempDir string) (*CommitFilesService, error) {
	return &CommitFilesService{
		adapter:      adapter,
		reposRoot:    reposRoot,
		reposTempDir: reposTempDir,
	}, nil
}

//nolint:funlen,gocognit // needs refactoring
func (s *CommitFilesService) CommitFiles(stream rpc.CommitFilesService_CommitFilesServer) error {
	ctx := stream.Context()
	headerRequest, err := stream.Recv()
	if err != nil {
		return ErrInternal(err)
	}

	header := headerRequest.GetHeader()
	if header == nil {
		return ErrInvalidArgument(types.ErrHeaderCannotBeEmpty)
	}

	base := header.GetBase()
	if base == nil {
		return ErrInvalidArgument(types.ErrBaseCannotBeEmpty)
	}

	committer := base.GetActor()
	if header.GetCommitter() != nil {
		committer = header.GetCommitter()
	}
	committerDate := time.Now().UTC()
	if header.GetAuthorDate() != 0 {
		committerDate = time.Unix(header.GetCommitterDate(), 0)
	}

	author := committer
	if header.GetAuthor() != nil {
		author = header.GetAuthor()
	}
	authorDate := committerDate
	if header.GetAuthorDate() != 0 {
		authorDate = time.Unix(header.GetAuthorDate(), 0)
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	// TODO: why are we using the giteat operations here?
	repo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		return processGitErrorf(err, "failed to open repo")
	}

	// check if repo is empty
	// IMPORTANT: we don't use gitea's repo.IsEmpty() as that only checks whether the default branch exists (in HEAD).
	// This can be an issue in case someone created a branch already in the repo (just default branch is missing).
	// In that case the user can accidentally create separate git histories (which most likely is unintended).
	// If the user wants to actually build a disconnected commit graph they can use the cli.
	isEmpty, err := repoHasBranches(ctx, repo)
	if err != nil {
		return ErrInternalf("failed to determine if repository is empty", err)
	}

	// ensure input data is valid
	if err = s.validateAndPrepareHeader(repo, isEmpty, header); err != nil {
		return err
	}

	// collect all file actions from grpc stream
	actions := make([]fileAction, 0, 16)
	if err = s.collectActions(stream, &actions); err != nil {
		return err
	}

	// create a new shared repo
	shared, err := NewSharedRepo(s.reposTempDir, base.GetRepoUid(), repo)
	if err != nil {
		return processGitErrorf(err, "failed to create shared repository")
	}
	defer shared.Close(ctx)

	// handle empty repo separately (as branch doesn't exist, no commit exists, ...)
	var parentCommitSHA string
	if isEmpty {
		err = s.prepareTreeEmptyRepo(ctx, shared, actions)
		if err != nil {
			return err
		}
	} else {
		parentCommitSHA, err = s.prepareTree(ctx, shared, header.GetBranchName(), actions)
		if err != nil {
			return err
		}
	}

	// Now write the tree
	treeHash, err := shared.WriteTree(ctx)
	if err != nil {
		return processGitErrorf(err, "failed to write tree object")
	}

	message := strings.TrimSpace(header.GetTitle())
	if len(header.GetMessage()) > 0 {
		message += "\n\n" + strings.TrimSpace(header.GetMessage())
	}
	// Now commit the tree
	commitSHA, err := shared.CommitTreeWithDate(
		ctx,
		parentCommitSHA,
		author,
		committer,
		treeHash,
		message,
		false,
		authorDate,
		committerDate,
	)
	if err != nil {
		return processGitErrorf(err, "failed to commit the tree")
	}

	if err = shared.PushCommitToBranch(ctx, base, commitSHA, header.GetNewBranchName()); err != nil {
		return processGitErrorf(err, "failed to push commits to remote repository")
	}

	commit, err := shared.GetCommit(commitSHA)
	if err != nil {
		return processGitErrorf(err, "failed to get commit for SHA %s", commitSHA)
	}

	return stream.SendAndClose(&rpc.CommitFilesResponse{
		CommitId: commit.ID.String(),
	})
}

func (s *CommitFilesService) prepareTree(ctx context.Context, shared *SharedRepo,
	branchName string, actions []fileAction) (string, error) {
	// clone original branch from repo
	if err := s.clone(ctx, shared, branchName); err != nil {
		return "", err
	}

	// Get the latest commit of the original branch
	commit, err := shared.GetBranchCommit(branchName)
	if err != nil {
		return "", processGitErrorf(err, "failed to get latest commit of the branch %s", branchName)
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

func (s *CommitFilesService) prepareTreeEmptyRepo(ctx context.Context, shared *SharedRepo,
	actions []fileAction) error {
	// init a new repo (full clone would cause risk that by time of push someone wrote to the remote repo!)
	err := shared.Init(ctx)
	if err != nil {
		return processGitErrorf(err, "failed to init shared tmp repository")
	}

	for _, action := range actions {
		if action.header.Action != rpc.CommitFilesActionHeader_CREATE {
			return ErrFailedPrecondition(types.ErrActionNotAllowedOnEmptyRepo)
		}

		filePath := files.CleanUploadFileName(action.header.GetPath())
		if filePath == "" {
			return ErrInvalidArgument(types.ErrInvalidPath)
		}

		reader := bytes.NewReader(action.content)
		if err = createFile(ctx, shared, nil, filePath, defaultFilePermission, reader); err != nil {
			return ErrInternalf("failed to create file '%s'", action.header.Path, err)
		}
	}

	return nil
}

func (s *CommitFilesService) validateAndPrepareHeader(repo *git.Repository, isEmpty bool,
	header *rpc.CommitFilesRequestHeader) error {
	if header.GetBranchName() == "" {
		defaultBranchRef, err := repo.GetDefaultBranch()
		if err != nil {
			return processGitErrorf(err, "failed to get default branch")
		}
		header.BranchName = defaultBranchRef
	}

	if header.GetNewBranchName() == "" {
		header.NewBranchName = header.GetBranchName()
	}

	// trim refs/heads/ prefixes to avoid issues when calling gitea API
	header.BranchName = strings.TrimPrefix(strings.TrimSpace(header.GetBranchName()), gitReferenceNamePrefixBranch)
	header.NewBranchName = strings.TrimPrefix(strings.TrimSpace(header.GetNewBranchName()), gitReferenceNamePrefixBranch)

	// if the repo is empty then we can skip branch existence checks
	if isEmpty {
		return nil
	}

	// ensure source branch exists
	if _, err := repo.GetBranch(header.GetBranchName()); err != nil {
		return processGitErrorf(err, "failed to get source branch %s", header.BranchName)
	}

	// ensure new branch doesn't exist yet (if new branch creation was requested)
	if header.GetBranchName() != header.GetNewBranchName() {
		existingBranch, err := repo.GetBranch(header.GetNewBranchName())
		if existingBranch != nil {
			return ErrAlreadyExistsf("branch %s already exists", existingBranch.Name)
		}
		if err != nil && !git.IsErrBranchNotExist(err) {
			return processGitErrorf(err, "failed to create new branch %s", header.NewBranchName)
		}
	}
	return nil
}

func (s *CommitFilesService) clone(
	ctx context.Context,
	shared *SharedRepo,
	branch string,
) error {
	if err := shared.Clone(ctx, branch); err != nil {
		return ErrInternalf("failed to clone branch '%s'", branch, err)
	}

	if err := shared.SetDefaultIndex(ctx); err != nil {
		return ErrInternalf("failed to set default index", err)
	}

	return nil
}

func (s *CommitFilesService) collectActions(
	stream rpc.CommitFilesService_CommitFilesServer,
	ptrActions *[]fileAction,
) error {
	if ptrActions == nil {
		return nil
	}
	actions := *ptrActions
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return ErrInternalf("receive request failed", err)
		}

		switch payload := req.GetAction().GetPayload().(type) {
		case *rpc.CommitFilesAction_Header:
			actions = append(actions, fileAction{header: payload.Header})
		case *rpc.CommitFilesAction_Content:
			if len(actions) == 0 {
				return ErrFailedPrecondition(types.ErrContentSentBeforeAction)
			}

			// append the content to the previous fileAction
			content := &actions[len(actions)-1].content
			*content = append(*content, payload.Content...)
		default:
			return ErrInternalf("unhandled fileAction payload type: %T", payload)
		}
	}
	if len(actions) == 0 {
		return ErrInvalidArgument(types.ErrActionListEmpty)
	}
	*ptrActions = actions
	return nil
}

func (s *CommitFilesService) processAction(
	ctx context.Context,
	shared *SharedRepo,
	action *fileAction,
	commit *git.Commit,
) (err error) {
	header := action.header
	if _, ok := rpc.CommitFilesActionHeader_ActionType_name[int32(header.Action)]; !ok {
		return ErrInvalidArgumentf("undefined file action %s", action.header.Action, types.ErrUndefinedAction)
	}

	filePath := files.CleanUploadFileName(header.GetPath())
	if filePath == "" {
		return ErrInvalidArgument(types.ErrInvalidPath)
	}

	reader := bytes.NewReader(action.content)

	switch header.Action {
	case rpc.CommitFilesActionHeader_CREATE:
		err = createFile(ctx, shared, commit, filePath, defaultFilePermission, reader)
	case rpc.CommitFilesActionHeader_UPDATE:
		err = updateFile(ctx, shared, commit, filePath, header.GetSha(), defaultFilePermission, reader)
	case rpc.CommitFilesActionHeader_MOVE:
		err = moveFile(ctx, shared, commit, filePath, defaultFilePermission, reader)
	case rpc.CommitFilesActionHeader_DELETE:
		err = deleteFile(ctx, shared, filePath)
	}

	return err
}

func createFile(ctx context.Context, repo *SharedRepo, commit *git.Commit,
	filePath, mode string, reader io.Reader) error {
	// only check path availability if a source commit is available (empty repo won't have such a commit)
	if commit != nil {
		if err := checkPathAvailability(commit, filePath, true); err != nil {
			return err
		}
	}

	hash, err := repo.WriteGitObject(ctx, reader)
	if err != nil {
		return processGitErrorf(err, "error hashing object")
	}

	// Add the object to the index
	if err = repo.AddObjectToIndex(ctx, mode, hash, filePath); err != nil {
		return processGitErrorf(err, "error creating object")
	}
	return nil
}

func updateFile(ctx context.Context, repo *SharedRepo, commit *git.Commit, filePath, sha,
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
		return processGitErrorf(err, "error hashing object")
	}

	if err = repo.AddObjectToIndex(ctx, mode, hash, filePath); err != nil {
		return processGitErrorf(err, "error updating object")
	}
	return nil
}

func moveFile(ctx context.Context, repo *SharedRepo, commit *git.Commit,
	filePath, mode string, reader io.Reader) error {
	buffer := &bytes.Buffer{}
	newPath, err := parsePayload(reader, buffer)
	if err != nil {
		return err
	}

	if buffer.Len() == 0 && newPath != "" {
		err = repo.ShowFile(ctx, filePath, commit.ID.String(), buffer)
		if err != nil {
			return processGitErrorf(err, "failed lookup for path %s", newPath)
		}
	}

	if err = checkPathAvailability(commit, newPath, false); err != nil {
		return err
	}

	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return processGitErrorf(err, "listing files error")
	}
	if !slices.Contains(filesInIndex, filePath) {
		return ErrNotFoundf("path %s not found", filePath)
	}

	hash, err := repo.WriteGitObject(ctx, buffer)
	if err != nil {
		return processGitErrorf(err, "error hashing object")
	}

	if err = repo.AddObjectToIndex(ctx, mode, hash, newPath); err != nil {
		return processGitErrorf(err, "add object error")
	}

	if err = repo.RemoveFilesFromIndex(ctx, filePath); err != nil {
		return processGitErrorf(err, "remove object error")
	}
	return nil
}

func deleteFile(ctx context.Context, repo *SharedRepo, filePath string) error {
	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return processGitErrorf(err, "listing files error")
	}
	if !slices.Contains(filesInIndex, filePath) {
		return ErrNotFoundf("file path %s not found", filePath)
	}

	if err = repo.RemoveFilesFromIndex(ctx, filePath); err != nil {
		return processGitErrorf(err, "remove object error")
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
		return nil, ErrNotFoundf("path %s not found", path)
	}
	if err != nil {
		return nil, processGitErrorf(err, "failed to get tree for path %s", path)
	}

	// If a SHA was given and the SHA given doesn't match the SHA of the fromTreePath, throw error
	if sha != "" && sha != entry.ID.String() {
		return nil, ErrInvalidArgumentf("sha does not match for path %s [given: %s, expected: %s]",
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
			return processGitErrorf(err, "failed to get tree entry for path %s", subTreePath)
		}
		switch {
		case index < len(parts)-1:
			if !entry.IsDir() {
				return ErrAlreadyExistsf("a file already exists where you're trying to create a subdirectory [path: %s]",
					subTreePath)
			}
		case entry.IsLink():
			return fmt.Errorf("a symbolic link %w where you're trying to create a subdirectory [path: %s]",
				types.ErrAlreadyExists, subTreePath)
		case entry.IsDir():
			return ErrAlreadyExistsf("a directory already exists where you're trying to create a subdirectory [path: %s]",
				subTreePath)
		case filePath != "" || isNewFile:
			return ErrAlreadyExistsf("file path %s already exists", filePath)
		}
	}
	return nil
}

// repoHasBranches returns true iff there's at least one branch in the repo (any branch)
// NOTE: This is different from repo.Empty(),
// as it doesn't care whether the existing branch is the default branch or not.
func repoHasBranches(ctx context.Context, repo *git.Repository) (bool, error) {
	// repo has branches IFF there's at least one commit that is reachable via a branch
	// (every existing branch points to a commit)
	stdout, _, runErr := git.NewCommand(ctx, "rev-list", "--max-count", "1", "--branches").
		RunStdBytes(&git.RunOpts{Dir: repo.Path})
	if runErr != nil {
		return false, processGitErrorf(runErr, "failed to trigger rev-list command")
	}

	return strings.TrimSpace(string(stdout)) == "", nil
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
