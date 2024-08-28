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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"
)

const (
	filePermissionDefault = "100644"
)

type FileAction string

const (
	CreateAction    FileAction = "CREATE"
	UpdateAction    FileAction = "UPDATE"
	DeleteAction    FileAction = "DELETE"
	MoveAction      FileAction = "MOVE"
	PatchTextAction FileAction = "PATCH_TEXT"
)

func (FileAction) Enum() []interface{} {
	return []interface{}{CreateAction, UpdateAction, DeleteAction, MoveAction, PatchTextAction}
}

// CommitFileAction holds file operation data.
type CommitFileAction struct {
	Action  FileAction
	Path    string
	Payload []byte
	SHA     sha.SHA
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
	CommitID sha.SHA
}

//nolint:gocognit,nestif
func (s *Service) CommitFiles(ctx context.Context, params *CommitFilesParams) (CommitFilesResponse, error) {
	if err := params.Validate(); err != nil {
		return CommitFilesResponse{}, err
	}

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

	// check if repo is empty

	// IMPORTANT: we don't use gitea's repo.IsEmpty() as that only checks whether the default branch exists (in HEAD).
	// This can be an issue in case someone created a branch already in the repo (just default branch is missing).
	// In that case the user can accidentally create separate git histories (which most likely is unintended).
	// If the user wants to actually build a disconnected commit graph they can use the cli.
	isEmpty, err := s.git.HasBranches(ctx, repoPath)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("CommitFiles: failed to determine if repository is empty: %w", err)
	}

	// validate and prepare input

	// ensure input data is valid
	// the commit will be nil for empty repositories
	commit, err := s.validateAndPrepareCommitFilesHeader(ctx, repoPath, isEmpty, params)
	if err != nil {
		return CommitFilesResponse{}, err
	}

	// ref updater
	var refOldSHA sha.SHA
	var refNewSHA sha.SHA

	branchRef := api.GetReferenceFromBranchName(params.Branch)
	if params.Branch != params.NewBranch {
		// we are creating a new branch, rather than updating the existing one
		refOldSHA = sha.Nil
		branchRef = api.GetReferenceFromBranchName(params.NewBranch)
	} else if commit != nil {
		refOldSHA = commit.SHA
	}

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath, branchRef)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to create ref updater: %w", err)
	}

	// run the actions in a shared repo

	err = sharedrepo.Run(ctx, refUpdater, s.tmpDir, repoPath, func(r *sharedrepo.SharedRepo) error {
		var parentCommits []sha.SHA
		var oldTreeSHA sha.SHA

		if isEmpty {
			oldTreeSHA = sha.EmptyTree
			err = s.prepareTreeEmptyRepo(ctx, r, params.Actions)
			if err != nil {
				return fmt.Errorf("failed to prepare empty tree: %w", err)
			}
		} else {
			parentCommits = append(parentCommits, commit.SHA)

			// get tree sha
			rootNode, err := s.git.GetTreeNode(ctx, repoPath, commit.SHA.String(), "")
			if err != nil {
				return fmt.Errorf("CommitFiles: failed to get original node: %w", err)
			}
			oldTreeSHA = rootNode.SHA

			err = r.SetIndex(ctx, commit.SHA)
			if err != nil {
				return fmt.Errorf("failed to set index in shared repository: %w", err)
			}

			err = s.prepareTree(ctx, r, commit.SHA, params.Actions)
			if err != nil {
				return fmt.Errorf("failed to prepare tree: %w", err)
			}
		}

		treeSHA, err := r.WriteTree(ctx)
		if err != nil {
			return fmt.Errorf("failed to write tree object: %w", err)
		}

		if oldTreeSHA.Equal(treeSHA) {
			return errors.InvalidArgument("No effective changes.")
		}

		message := strings.TrimSpace(params.Title)
		if len(params.Message) > 0 {
			message += "\n\n" + strings.TrimSpace(params.Message)
		}

		authorSig := &api.Signature{
			Identity: api.Identity{
				Name:  author.Name,
				Email: author.Email,
			},
			When: authorDate,
		}

		committerSig := &api.Signature{
			Identity: api.Identity{
				Name:  committer.Name,
				Email: committer.Email,
			},
			When: committerDate,
		}

		commitSHA, err := r.CommitTree(ctx, authorSig, committerSig, treeSHA, message, false, parentCommits...)
		if err != nil {
			return fmt.Errorf("failed to commit the tree: %w", err)
		}

		refNewSHA = commitSHA

		if err := refUpdater.Init(ctx, refOldSHA, refNewSHA); err != nil {
			return fmt.Errorf("failed to init ref updater old=%s new=%s: %w", refOldSHA, refNewSHA, err)
		}

		return nil
	})
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("CommitFiles: failed to create commit in shared repository: %w", err)
	}

	// get commit

	commit, err = s.git.GetCommit(ctx, repoPath, refNewSHA.String())
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to get commit for SHA %s: %w",
			refNewSHA.String(), err)
	}

	return CommitFilesResponse{
		CommitID: commit.SHA,
	}, nil
}

func (s *Service) prepareTree(
	ctx context.Context,
	r *sharedrepo.SharedRepo,
	treeishSHA sha.SHA,
	actions []CommitFileAction,
) error {
	// patch file actions are executed in batch for a single file
	patchMap := map[string][]*CommitFileAction{}

	// keep track of what paths have been written to detect conflicting actions
	modifiedPaths := map[string]bool{}

	for i := range actions {
		act := &actions[i]

		// patch text actions are executed in per-file batches.
		if act.Action == PatchTextAction {
			patchMap[act.Path] = append(patchMap[act.Path], act)
			continue
		}
		// anything else is executed as is
		modifiedPath, err := s.processAction(ctx, r, treeishSHA, act)
		if err != nil {
			return fmt.Errorf("failed to process action %s on %q: %w", act.Action, act.Path, err)
		}

		if modifiedPaths[modifiedPath] {
			return errors.InvalidArgument("More than one conflicting actions are modifying file %q.", modifiedPath)
		}
		modifiedPaths[modifiedPath] = true
	}

	for filePath, patchActions := range patchMap {
		// combine input across actions
		var fileSHA sha.SHA
		var payloads [][]byte
		for _, act := range patchActions {
			payloads = append(payloads, act.Payload)
			if fileSHA.IsEmpty() {
				fileSHA = act.SHA
				continue
			}

			// there can only be one file sha for a given path and commit.
			if !act.SHA.IsEmpty() && !fileSHA.Equal(act.SHA) {
				return errors.InvalidArgument(
					"patch text actions for %q contain different SHAs %q and %q",
					filePath,
					act.SHA,
					fileSHA,
				)
			}
		}

		if err := r.PatchTextFile(ctx, treeishSHA, filePath, fileSHA, payloads); err != nil {
			return fmt.Errorf("failed to process action %s on %q: %w", PatchTextAction, filePath, err)
		}

		if modifiedPaths[filePath] {
			return errors.InvalidArgument("More than one conflicting action are modifying file %q.", filePath)
		}
		modifiedPaths[filePath] = true
	}

	return nil
}

func (s *Service) prepareTreeEmptyRepo(
	ctx context.Context,
	r *sharedrepo.SharedRepo,
	actions []CommitFileAction,
) error {
	for _, action := range actions {
		if action.Action != CreateAction {
			return errors.PreconditionFailed("action not allowed on empty repository")
		}

		filePath := api.CleanUploadFileName(action.Path)
		if filePath == "" {
			return errors.InvalidArgument("invalid path")
		}

		if err := r.CreateFile(ctx, sha.None, filePath, filePermissionDefault, action.Payload); err != nil {
			return errors.Internal(err, "failed to create file '%s'", action.Path)
		}
	}

	return nil
}

func (s *Service) validateAndPrepareCommitFilesHeader(
	ctx context.Context,
	repoPath string,
	isEmpty bool,
	params *CommitFilesParams,
) (*api.Commit, error) {
	if params.Branch == "" {
		defaultBranchRef, err := s.git.GetDefaultBranch(ctx, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get default branch: %w", err)
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
		return nil, nil //nolint:nilnil // an empty repository has no commit and there's no error
	}

	// ensure source branch exists
	branch, err := s.git.GetBranch(ctx, repoPath, params.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get source branch '%s': %w", params.Branch, err)
	}

	// ensure new branch doesn't exist yet (if new branch creation was requested)
	if params.Branch != params.NewBranch {
		existingBranch, err := s.git.GetBranch(ctx, repoPath, params.NewBranch)
		if existingBranch != nil {
			return nil, errors.Conflict("branch %s already exists", existingBranch.Name)
		}
		if err != nil && !errors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to create new branch '%s': %w", params.NewBranch, err)
		}
	}

	return branch.Commit, nil
}

func (s *Service) processAction(
	ctx context.Context,
	r *sharedrepo.SharedRepo,
	treeishSHA sha.SHA,
	action *CommitFileAction,
) (modifiedPath string, err error) {
	filePath := api.CleanUploadFileName(action.Path)
	if filePath == "" {
		return "", errors.InvalidArgument("path cannot be empty")
	}
	modifiedPath = filePath
	switch action.Action {
	case CreateAction:
		err = r.CreateFile(ctx, treeishSHA, filePath, filePermissionDefault, action.Payload)
	case UpdateAction:
		err = r.UpdateFile(ctx, treeishSHA, filePath, action.SHA, filePermissionDefault, action.Payload)
	case MoveAction:
		modifiedPath, err = r.MoveFile(ctx, treeishSHA, filePath, action.SHA, filePermissionDefault, action.Payload)
	case DeleteAction:
		err = r.DeleteFile(ctx, filePath)
	case PatchTextAction:
		return "", fmt.Errorf("action %s not supported by this method", action.Action)
	default:
		err = fmt.Errorf("unknown file action %q", action.Action)
	}

	return modifiedPath, err
}

/*
func (s *Service) prepareTree(
	ctx context.Context,
	shared *api.SharedRepo,
	actions []CommitFileAction,
	commit *api.Commit,
) error {
	// execute all actions
	for i := range actions {
		if err := s.processAction(ctx, shared, &actions[i], commit); err != nil {
			return err
		}
	}

	return nil
}

func prepareTreeEmptyRepo(
	ctx context.Context,
	shared *api.SharedRepo,
	actions []CommitFileAction,
) error {
	for _, action := range actions {
		if action.Action != CreateAction {
			return errors.PreconditionFailed("action not allowed on empty repository")
		}

		filePath := api.CleanUploadFileName(action.Path)
		if filePath == "" {
			return errors.InvalidArgument("invalid path")
		}

		if err := createFile(ctx, shared, nil, filePath, defaultFilePermission, action.Payload); err != nil {
			return errors.Internal(err, "failed to create file '%s'", action.Path)
		}
	}

	return nil
}

func (s *Service) processAction(
	ctx context.Context,
	shared *api.SharedRepo,
	action *CommitFileAction,
	commit *api.Commit,
) (err error) {
	filePath := api.CleanUploadFileName(action.Path)
	if filePath == "" {
		return errors.InvalidArgument("path cannot be empty")
	}

	switch action.Action {
	case CreateAction:
		err = createFile(ctx, shared, commit, filePath, defaultFilePermission, action.Payload)
	case UpdateAction:
		err = updateFile(ctx, shared, commit, filePath, action.SHA, defaultFilePermission, action.Payload)
	case MoveAction:
		err = moveFile(ctx, shared, commit, filePath, action.SHA, defaultFilePermission, action.Payload)
	case DeleteAction:
		err = deleteFile(ctx, shared, filePath)
	}

	return err
}

func createFile(ctx context.Context, repo *api.SharedRepo, commit *api.Commit,
	filePath, mode string, payload []byte) error {
	// only check path availability if a source commit is available (empty repo won't have such a commit)
	if commit != nil {
		if err := checkPathAvailability(ctx, repo, commit, filePath, true); err != nil {
			return err
		}
	}

	hash, err := repo.WriteGitObject(ctx, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("createFile: error hashing object: %w", err)
	}

	// Add the object to the index
	if err = repo.AddObjectToIndex(ctx, mode, hash.String(), filePath); err != nil {
		return fmt.Errorf("createFile: error creating object: %w", err)
	}
	return nil
}

func updateFile(
	ctx context.Context,
	repo *api.SharedRepo,
	commit *api.Commit,
	filePath string,
	sha string,
	mode string,
	payload []byte,
) error {
	// get file mode from existing file (default unless executable)
	entry, err := getFileEntry(ctx, repo, commit, sha, filePath)
	if err != nil {
		return err
	}
	if entry.IsExecutable() {
		mode = "100755"
	}

	hash, err := repo.WriteGitObject(ctx, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("updateFile: error hashing object: %w", err)
	}

	if err = repo.AddObjectToIndex(ctx, mode, hash.String(), filePath); err != nil {
		return fmt.Errorf("updateFile: error updating object: %w", err)
	}
	return nil
}

func moveFile(
	ctx context.Context,
	repo *api.SharedRepo,
	commit *api.Commit,
	filePath string,
	sha string,
	mode string,
	payload []byte,
) error {
	newPath, newContent, err := parseMovePayload(payload)
	if err != nil {
		return err
	}

	// ensure file exists and matches SHA
	entry, err := getFileEntry(ctx, repo, commit, sha, filePath)
	if err != nil {
		return err
	}

	// ensure new path is available
	if err = checkPathAvailability(ctx, repo, commit, newPath, false); err != nil {
		return err
	}

	var fileHash string
	var fileMode string
	if newContent != nil {
		hash, err := repo.WriteGitObject(ctx, bytes.NewReader(newContent))
		if err != nil {
			return fmt.Errorf("moveFile: error hashing object: %w", err)
		}

		fileHash = hash.String()
		fileMode = mode
		if entry.IsExecutable() {
			fileMode = "100755"
		}
	} else {
		fileHash = entry.SHA.String()
		fileMode = entry.Mode.String()
	}

	if err = repo.AddObjectToIndex(ctx, fileMode, fileHash, newPath); err != nil {
		return fmt.Errorf("moveFile: add object error: %w", err)
	}

	if err = repo.RemoveFilesFromIndex(ctx, filePath); err != nil {
		return fmt.Errorf("moveFile: remove object error: %w", err)
	}
	return nil
}

func deleteFile(ctx context.Context, repo *api.SharedRepo, filePath string) error {
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
	ctx context.Context,
	repo *api.SharedRepo,
	commit *api.Commit,
	sha string,
	path string,
) (*api.TreeNode, error) {
	entry, err := repo.GetTreeNode(ctx, commit.SHA.String(), path)
	if errors.IsNotFound(err) {
		return nil, errors.NotFound("path %s not found", path)
	}
	if err != nil {
		return nil, fmt.Errorf("getFileEntry: failed to get tree for path %s: %w", path, err)
	}

	// If a SHA was given and the SHA given doesn't match the SHA of the fromTreePath, throw error
	if sha != "" && sha != entry.SHA.String() {
		return nil, errors.InvalidArgument("sha does not match for path %s [given: %s, expected: %s]",
			path, sha, entry.SHA)
	}

	return entry, nil
}

// checkPathAvailability ensures that the path is available for the requested operation.
// For the path where this file will be created/updated, we need to make
// sure no parts of the path are existing files or links except for the last
// item in the path which is the file name, and that shouldn't exist IF it is
// a new file OR is being moved to a new path.
func checkPathAvailability(
	ctx context.Context,
	repo *api.SharedRepo,
	commit *api.Commit,
	filePath string,
	isNewFile bool,
) error {
	parts := strings.Split(filePath, "/")
	subTreePath := ""
	for index, part := range parts {
		subTreePath = path.Join(subTreePath, part)
		entry, err := repo.GetTreeNode(ctx, commit.SHA.String(), subTreePath)
		if err != nil {
			if errors.IsNotFound(err) {
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
			return errors.Conflict("a symbolic link already exist where you're trying to create a subdirectory [path: %s]",
				subTreePath)
		case entry.IsDir():
			return errors.Conflict("a directory already exists where you're trying to create a subdirectory [path: %s]",
				subTreePath)
		case filePath != "" || isNewFile:
			return errors.Conflict("file path %s already exists", filePath)
		}
	}
	return nil
}

func parseMovePayload(payload []byte) (string, []byte, error) {
	var newContent []byte
	var newPath string
	filePathEnd := bytes.IndexByte(payload, 0)
	if filePathEnd < 0 {
		newPath = string(payload)
		newContent = nil
	} else {
		newPath = string(payload[:filePathEnd])
		newContent = payload[filePathEnd+1:]
	}

	newPath = api.CleanUploadFileName(newPath)
	if newPath == "" {
		return "", nil, api.ErrInvalidPath
	}

	return newPath, newContent, nil
}
*/
