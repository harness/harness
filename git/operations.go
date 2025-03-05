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
	"github.com/harness/gitness/git/parser"
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
	Title     string // Deprecated
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

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath)
	if err != nil {
		return CommitFilesResponse{}, fmt.Errorf("failed to create ref updater: %w", err)
	}

	// run the actions in a shared repo

	err = sharedrepo.Run(ctx, refUpdater, s.sharedRepoRoot, repoPath, func(r *sharedrepo.SharedRepo) error {
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

		var message string
		if params.Title != "" {
			// Title is deprecated and should not be sent, but if it's sent assume we need to generate the full message.
			message = parser.CleanUpWhitespace(CommitMessage(params.Title, params.Message))
		} else {
			message = parser.CleanUpWhitespace(params.Message)
		}

		commitSHA, err := r.CommitTree(ctx, authorSig, committerSig, treeSHA, message, false, parentCommits...)
		if err != nil {
			return fmt.Errorf("failed to commit the tree: %w", err)
		}

		refNewSHA = commitSHA

		ref := hook.ReferenceUpdate{
			Ref: branchRef,
			Old: refOldSHA,
			New: refNewSHA,
		}

		if err := refUpdater.Init(ctx, []hook.ReferenceUpdate{ref}); err != nil {
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
