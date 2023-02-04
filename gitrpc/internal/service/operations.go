// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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

//nolint:funlen // needs refactoring
func (s *CommitFilesService) CommitFiles(stream rpc.CommitFilesService_CommitFilesServer) error {
	ctx := stream.Context()
	headerRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	header := headerRequest.GetHeader()
	if header == nil {
		return types.ErrHeaderCannotBeEmpty
	}

	base := header.GetBase()
	if base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	committer := base.GetActor()
	if header.GetCommitter() != nil {
		committer = header.GetCommitter()
	}
	author := committer
	if header.GetAuthor() != nil {
		author = header.GetAuthor()
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	// TODO: why are we using the giteat operations here?
	repo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		return err
	}

	// check if repo is empty
	// IMPORTANT: we don't use gitea's repo.IsEmpty() as that only checks whether the default branch exists (in HEAD).
	// This can be an issue in case someone created a branch already in the repo (just default branch is missing).
	// In that case the user can accidentaly create separate git histories (which most likely is unintended).
	// If the user wants to actually build a disconnected commit graph they can use the cli.
	isEmpty, err := repoHasBranches(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to determine if repo is empty: %w", err)
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
		return err
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
		return err
	}

	message := strings.TrimSpace(header.GetTitle())
	if len(header.GetMessage()) > 0 {
		message += "\n\n" + strings.TrimSpace(header.GetMessage())
	}
	// Now commit the tree
	commitSHA, err := shared.CommitTree(ctx, parentCommitSHA, author, committer, treeHash, message, false)
	if err != nil {
		return err
	}

	if err = shared.PushCommit(ctx, base, commitSHA, header.GetNewBranchName()); err != nil {
		return err
	}

	commit, err := shared.GetCommit(commitSHA)
	if err != nil {
		return err
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
		return "", err
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
		return fmt.Errorf("failed to init shared tmp repo: %w", err)
	}

	for _, action := range actions {
		if action.header.Action != rpc.CommitFilesActionHeader_CREATE {
			return types.ErrActionNotAllowedOnEmptyRepo
		}

		filePath := files.CleanUploadFileName(action.header.GetPath())
		if filePath == "" {
			return types.ErrInvalidPath
		}

		reader := bytes.NewReader(action.content)
		if err = createFile(ctx, shared, nil, filePath, defaultFilePermission, reader); err != nil {
			return fmt.Errorf("failed to create file '%s': %w", action.header.Path, err)
		}
	}

	return nil
}

func (s *CommitFilesService) validateAndPrepareHeader(repo *git.Repository, isEmpty bool,
	header *rpc.CommitFilesRequestHeader) error {
	if header.GetBranchName() == "" {
		defaultBranchRef, err := repo.GetDefaultBranch()
		if err != nil {
			return err
		}
		header.BranchName = defaultBranchRef
	}

	if header.GetNewBranchName() == "" {
		header.NewBranchName = header.GetBranchName()
	}

	// trim refs/heads/ prefixes to avoid issues when calling gitea API
	header.BranchName = strings.TrimPrefix(strings.TrimSpace(header.GetBranchName()), gitReferenceNamePrefixBranch)
	header.NewBranchName = strings.TrimPrefix(strings.TrimSpace(header.GetNewBranchName()), gitReferenceNamePrefixBranch)

	// if the repo is empty then we can skip branch existance checks
	if isEmpty {
		return nil
	}

	// ensure source branch exists
	if _, err := repo.GetBranch(header.GetBranchName()); err != nil {
		return err
	}

	// ensure new branch doesn't exist yet (if new branch creation was requested)
	if header.GetBranchName() != header.GetNewBranchName() {
		existingBranch, err := repo.GetBranch(header.GetNewBranchName())
		if existingBranch != nil {
			return fmt.Errorf("branch %s %w", existingBranch.Name, types.ErrAlreadyExists)
		}
		if err != nil && !git.IsErrBranchNotExist(err) {
			return err
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
		return fmt.Errorf("failed to clone branch '%s': %w", branch, err)
	}

	if err := shared.SetDefaultIndex(ctx); err != nil {
		return fmt.Errorf("failed to set default index: %w", err)
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

			return fmt.Errorf("receive request: %w", err)
		}

		switch payload := req.GetAction().GetPayload().(type) {
		case *rpc.CommitFilesAction_Header:
			actions = append(actions, fileAction{header: payload.Header})
		case *rpc.CommitFilesAction_Content:
			if len(actions) == 0 {
				return types.ErrContentSentBeforeAction
			}

			// append the content to the previous fileAction
			content := &actions[len(actions)-1].content
			*content = append(*content, payload.Content...)
		default:
			return fmt.Errorf("unhandled fileAction payload type: %T", payload)
		}
	}
	if len(actions) == 0 {
		return types.ErrActionListEmpty
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
	defer func() {
		if err != nil {
			err = fmt.Errorf("in processActions: %w", err)
		}
	}()

	header := action.header
	if _, ok := rpc.CommitFilesActionHeader_ActionType_name[int32(header.Action)]; !ok {
		return fmt.Errorf("%s %w", action.header.Action, types.ErrUndefinedAction)
	}

	filePath := files.CleanUploadFileName(header.GetPath())
	if filePath == "" {
		return types.ErrInvalidPath
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
		return fmt.Errorf("error hashing object %w", err)
	}

	// Add the object to the index
	if err = repo.AddObjectToIndex(ctx, mode, hash, filePath); err != nil {
		return fmt.Errorf("error creating object: %w", err)
	}
	return nil
}

func updateFile(ctx context.Context, repo *SharedRepo, commit *git.Commit, filePath, sha,
	mode string, reader io.Reader) error {
	// get file mode from existing file (default unless executable)
	entry, err := getFileEntry(commit, sha, filePath)
	if err != nil {
		return fmt.Errorf("failed to get file entry: %w", err)
	}
	if entry.IsExecutable() {
		mode = "100755"
	}

	hash, err := repo.WriteGitObject(ctx, reader)
	if err != nil {
		return fmt.Errorf("error hashing object %w", err)
	}

	if err = repo.AddObjectToIndex(ctx, mode, hash, filePath); err != nil {
		return fmt.Errorf("error updating object: %w", err)
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
			return err
		}
	}

	if err = checkPathAvailability(commit, newPath, false); err != nil {
		return err
	}

	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return fmt.Errorf("listing files error %w", err)
	}
	if !slices.Contains(filesInIndex, filePath) {
		return fmt.Errorf("%s %w", filePath, types.ErrNotFound)
	}

	hash, err := repo.WriteGitObject(ctx, buffer)
	if err != nil {
		return fmt.Errorf("error hashing object %w", err)
	}

	if err = repo.AddObjectToIndex(ctx, mode, hash, newPath); err != nil {
		return fmt.Errorf("add object: %w", err)
	}

	if err = repo.RemoveFilesFromIndex(ctx, filePath); err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

func deleteFile(ctx context.Context, repo *SharedRepo, filePath string) error {
	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return fmt.Errorf("listing files error %w", err)
	}
	if !slices.Contains(filesInIndex, filePath) {
		return fmt.Errorf("%s %w", filePath, types.ErrNotFound)
	}

	if err = repo.RemoveFilesFromIndex(ctx, filePath); err != nil {
		return fmt.Errorf("remove object: %w", err)
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
		return nil, fmt.Errorf("%s %w", path, types.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	// If a SHA was given and the SHA given doesn't match the SHA of the fromTreePath, throw error
	if sha == "" || sha != entry.ID.String() {
		return nil, fmt.Errorf("%w for path %s [given: %s, expected: %s]",
			types.ErrSHADoesNotMatch, path, sha, entry.ID.String())
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
			return err
		}
		switch {
		case index < len(parts)-1:
			if !entry.IsDir() {
				return fmt.Errorf("a file %w where you're trying to create a subdirectory [path: %s]",
					types.ErrAlreadyExists, subTreePath)
			}
		case entry.IsLink():
			return fmt.Errorf("a symbolic link %w where you're trying to create a subdirectory [path: %s]",
				types.ErrAlreadyExists, subTreePath)
		case entry.IsDir():
			return fmt.Errorf("a directory %w where you're trying to create a subdirectory [path: %s]",
				types.ErrAlreadyExists, subTreePath)
		case filePath != "" || isNewFile:
			return fmt.Errorf("%s %w", filePath, types.ErrAlreadyExists)
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
		return false, fmt.Errorf("failed to trigger rev-list command: %w", runErr)
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
