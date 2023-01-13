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
	filePrefix = "file://"
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
	author := header.GetAuthor()
	// in case no explicit author is provided use actor as author.
	if author == nil {
		author = committer
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	// TODO: why are we using the giteat operations here?
	repo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		return err
	}

	if err = s.validateHeader(repo, header); err != nil {
		return err
	}

	actions := make([]fileAction, 0, 16)
	if err = s.collectActions(stream, &actions); err != nil {
		return err
	}

	// create a shared repo
	shared, err := NewSharedRepo(s.reposTempDir, base.GetRepoUid(), repo)
	if err != nil {
		return err
	}
	defer shared.Close(ctx)

	if err = s.clone(ctx, repo, shared, header.GetBranchName()); err != nil {
		return err
	}

	// Get the commit of the original branch
	commit, err := shared.GetBranchCommit(header.GetBranchName())
	if err != nil {
		return err
	}

	for _, action := range actions {
		action := action
		if err = s.processAction(ctx, shared, &action, commit); err != nil {
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
	commitHash, err := shared.CommitTree(ctx, commit.ID.String(), author, committer, treeHash, message, false)
	if err != nil {
		return err
	}

	if err = shared.Push(ctx, base, commitHash, header.GetNewBranchName()); err != nil {
		return err
	}

	commit, err = shared.GetCommit(commitHash)
	if err != nil {
		return err
	}

	return stream.SendAndClose(&rpc.CommitFilesResponse{
		CommitId: commit.ID.String(),
	})
}

func (s *CommitFilesService) validateHeader(repo *git.Repository, header *rpc.CommitFilesRequestHeader) error {
	if header.GetBranchName() == "" {
		branch, err := repo.GetDefaultBranch()
		if err != nil {
			return err
		}
		header.BranchName = branch
	}

	if header.GetNewBranchName() == "" {
		header.NewBranchName = header.GetBranchName()
	}

	if _, err := repo.GetBranch(header.GetBranchName()); err != nil {
		return err
	}

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
	repo *git.Repository,
	shared *SharedRepo,
	branch string,
) error {
	if err := shared.Clone(ctx, branch); err != nil {
		empty, _ := repo.IsEmpty()
		if !git.IsErrBranchNotExist(err) || !empty {
			return err
		}
		if errInit := shared.Init(ctx); errInit != nil {
			return errInit
		}
		return err
	}
	return shared.SetDefaultIndex(ctx)
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

	mode := "100644" // 0o644 default file permission
	reader := bytes.NewReader(action.content)

	switch header.Action {
	case rpc.CommitFilesActionHeader_CREATE:
		err = createFile(ctx, shared, commit, filePath, mode, reader)
	case rpc.CommitFilesActionHeader_UPDATE:
		err = updateFile(ctx, shared, commit, filePath, header.GetSha(), mode, reader)
	case rpc.CommitFilesActionHeader_MOVE:
		err = moveFile(ctx, shared, commit, filePath, mode, reader)
	case rpc.CommitFilesActionHeader_DELETE:
		err = deleteFile(ctx, shared, filePath)
	}

	return err
}

func createFile(ctx context.Context, repo *SharedRepo, commit *git.Commit, filePath,
	mode string, reader io.Reader) error {
	if err := checkPath(commit, filePath, true); err != nil {
		return err
	}
	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return fmt.Errorf("listing files error %w", err)
	}
	if slices.Contains(filesInIndex, filePath) {
		return fmt.Errorf("%s %w", filePath, types.ErrAlreadyExists)
	}
	hash, err := repo.HashObject(ctx, reader)
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
	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return fmt.Errorf("listing files error %w", err)
	}
	if !slices.Contains(filesInIndex, filePath) {
		return fmt.Errorf("%s %w", filePath, types.ErrNotFound)
	}
	if commit != nil {
		var entry *git.TreeEntry
		entry, err = getFileEntry(commit, sha, filePath)
		if err != nil {
			return err
		}
		if entry.IsExecutable() {
			mode = "100755"
		}
	}
	hash, err := repo.HashObject(ctx, reader)
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

	if err = checkPath(commit, newPath, false); err != nil {
		return err
	}
	filesInIndex, err := repo.LsFiles(ctx, filePath)
	if err != nil {
		return fmt.Errorf("listing files error %w", err)
	}
	if !slices.Contains(filesInIndex, filePath) {
		return fmt.Errorf("%s %w", filePath, types.ErrNotFound)
	}
	if slices.Contains(filesInIndex, newPath) {
		return fmt.Errorf("%s %w", filePath, types.ErrAlreadyExists)
	}
	hash, err := repo.HashObject(ctx, buffer)
	if err != nil {
		return fmt.Errorf("error hashing object %w", err)
	}
	if err = repo.AddObjectToIndex(ctx, mode, hash, newPath); err != nil {
		return fmt.Errorf("created object: %w", err)
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

func checkPath(commit *git.Commit, filePath string, isNewFile bool) error {
	// For the path where this file will be created/updated, we need to make
	// sure no parts of the path are existing files or links except for the last
	// item in the path which is the file name, and that shouldn't exist IF it is
	// a new file OR is being moved to a new path.
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
