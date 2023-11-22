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

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/adapter"
	"github.com/harness/gitness/git/check"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
)

type BranchSortOption int

const (
	BranchSortOptionDefault BranchSortOption = iota
	BranchSortOptionName
	BranchSortOptionDate
)

var listBranchesRefFields = []types.GitReferenceField{
	types.GitReferenceFieldRefName,
	types.GitReferenceFieldObjectName,
}

type Branch struct {
	Name   string
	SHA    string
	Commit *Commit
}

type CreateBranchParams struct {
	WriteParams
	// BranchName is the name of the branch
	BranchName string
	// Target is a git reference (branch / tag / commit SHA)
	Target string
}

type CreateBranchOutput struct {
	Branch Branch
}

type GetBranchParams struct {
	ReadParams
	// BranchName is the name of the branch
	BranchName string
}

type GetBranchOutput struct {
	Branch Branch
}

type DeleteBranchParams struct {
	WriteParams
	// Name is the name of the branch
	BranchName string
}

type ListBranchesParams struct {
	ReadParams
	IncludeCommit bool
	Query         string
	Sort          BranchSortOption
	Order         SortOrder
	Page          int32
	PageSize      int32
}

type ListBranchesOutput struct {
	Branches []Branch
}

func (s *Service) CreateBranch(ctx context.Context, params *CreateBranchParams) (*CreateBranchOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	env := CreateEnvironmentForPush(ctx, params.WriteParams)

	if err := check.BranchName(params.BranchName); err != nil {
		return nil, errors.InvalidArgument(err.Error())
	}

	repo, err := s.adapter.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	if ok, err := repo.IsEmpty(); ok {
		if err != nil {
			return nil, errors.Internal("failed to check if repository is empty: %v", err)
		}
		return nil, errors.InvalidArgument("branch cannot be created on empty repository")
	}

	sharedRepo, err := s.adapter.SharedRepository(s.tmpDir, params.RepoUID, repo.Path)
	if err != nil {
		return nil, errors.Internal("failed to create new shared repo", err)
	}
	defer sharedRepo.Close(ctx)

	// clone repo (with HEAD branch - target might be anything)
	err = sharedRepo.Clone(ctx, "")
	if err != nil {
		return nil, errors.Internal("failed to clone shared repo with branch '%s'", params.BranchName, err)
	}

	_, err = sharedRepo.GetBranchCommit(params.BranchName)
	// return an error if branch alredy exists (push doesn't fail if it's a noop or fast forward push)
	if err == nil {
		return nil, errors.Conflict("branch '%s' already exists", params.BranchName)
	}
	if !gitea.IsErrNotExist(err) {
		return nil, errors.Internal("branch creation of '%s' failed: %w", params.BranchName, err)
	}

	// get target commit (as target could be branch/tag/commit, and tag can't be pushed using source:destination syntax)
	targetCommit, err := s.adapter.GetCommit(ctx, sharedRepo.Path(), strings.TrimSpace(params.Target))
	if gitea.IsErrNotExist(err) {
		return nil, errors.NotFound("target '%s' doesn't exist", params.Target)
	}
	if err != nil {
		return nil, errors.Internal("failed to get commit id for target '%s'", params.Target, err)
	}

	// push to new branch (all changes should go through push flow for hooks and other safety meassures)
	err = sharedRepo.PushCommitToBranch(ctx, targetCommit.SHA, params.BranchName, false, env...)
	if err != nil {
		return nil, err
	}

	// get branch
	// TODO: get it from shared repo to avoid opening another gitea repo and having to strip here.
	gitBranch, err := s.adapter.GetBranch(
		ctx,
		repoPath,
		strings.TrimPrefix(params.BranchName, gitReferenceNamePrefixBranch),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get git branch '%s': %w", params.BranchName, err)
	}

	branch, err := mapBranch(gitBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc branch %v: %w", gitBranch.Name, err)
	}

	return &CreateBranchOutput{
		Branch: *branch,
	}, nil
}

func (s *Service) GetBranch(ctx context.Context, params *GetBranchParams) (*GetBranchOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	sanitizedBranchName := strings.TrimPrefix(params.BranchName, gitReferenceNamePrefixBranch)

	gitBranch, err := s.adapter.GetBranch(ctx, repoPath, sanitizedBranchName)
	if err != nil {
		return nil, err
	}

	branch, err := mapBranch(gitBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc branch %v: %w", gitBranch.Name, err)
	}

	return &GetBranchOutput{
		Branch: *branch,
	}, nil
}

func (s *Service) DeleteBranch(ctx context.Context, params *DeleteBranchParams) error {
	if params == nil {
		return ErrNoParamsProvided
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	env := CreateEnvironmentForPush(ctx, params.WriteParams)

	repo, err := s.adapter.OpenRepository(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	sharedRepo, err := s.adapter.SharedRepository(s.tmpDir, params.RepoUID, repo.Path)
	if err != nil {
		return fmt.Errorf("failed to create new shared repo: %w", err)
	}
	defer sharedRepo.Close(ctx)

	// clone repo (technically we don't care about which branch we clone)
	err = sharedRepo.Clone(ctx, params.BranchName)
	if err != nil {
		return fmt.Errorf("failed to clone shared repo with branch '%s': %w", params.BranchName, err)
	}

	// get latest branch commit before we delete
	_, err = sharedRepo.GetBranchCommit(params.BranchName)
	if err != nil {
		return fmt.Errorf("failed to get gitea commit for branch '%s': %w", params.BranchName, err)
	}

	// push to remote (all changes should go through push flow for hooks and other safety meassures)
	// NOTE: setting sourceRef to empty will delete the remote branch when pushing:
	// https://git-scm.com/docs/git-push#Documentation/git-push.txt-ltrefspecgt82308203
	err = sharedRepo.PushDeleteBranch(ctx, params.BranchName, true, env...)
	if err != nil {
		return fmt.Errorf("failed to delete branch '%s' from remote repo: %w", params.BranchName, err)
	}

	return nil
}

func (s *Service) ListBranches(ctx context.Context, params *ListBranchesParams) (*ListBranchesOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	gitBranches, err := s.listBranchesLoadReferenceData(ctx, repoPath, types.BranchFilter{
		IncludeCommit: params.IncludeCommit,
		Query:         params.Query,
		Sort:          mapBranchesSortOption(params.Sort),
		Order:         mapToSortOrder(params.Order),
		Page:          params.Page,
		PageSize:      params.PageSize,
	})
	if err != nil {
		return nil, err
	}

	// get commits if needed (single call for perf savings: 1s-4s vs 5s-20s)
	if params.IncludeCommit {
		commitSHAs := make([]string, len(gitBranches))
		for i := range gitBranches {
			commitSHAs[i] = gitBranches[i].SHA
		}

		var gitCommits []types.Commit
		gitCommits, err = s.adapter.GetCommits(ctx, repoPath, commitSHAs)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit: %w", err)
		}

		for i := range gitCommits {
			gitBranches[i].Commit = &gitCommits[i]
		}
	}

	branches := make([]Branch, len(gitBranches))
	for i, branch := range gitBranches {
		b, err := mapBranch(branch)
		if err != nil {
			return nil, err
		}
		branches[i] = *b
	}

	return &ListBranchesOutput{
		Branches: branches,
	}, nil
}

func (s *Service) listBranchesLoadReferenceData(
	ctx context.Context,
	repoPath string,
	filter types.BranchFilter,
) ([]*types.Branch, error) {
	// TODO: can we be smarter with slice allocation
	branches := make([]*types.Branch, 0, 16)
	handler := listBranchesWalkReferencesHandler(&branches)
	instructor, endsAfter, err := wrapInstructorWithOptionalPagination(
		adapter.DefaultInstructor, // branches only have one target type, default instructor is enough
		filter.Page,
		filter.PageSize,
	)
	if err != nil {
		return nil, errors.InvalidArgument("invalid pagination details: %v", err)
	}

	opts := &types.WalkReferencesOptions{
		Patterns:   createReferenceWalkPatternsFromQuery(gitReferenceNamePrefixBranch, filter.Query),
		Sort:       filter.Sort,
		Order:      filter.Order,
		Fields:     listBranchesRefFields,
		Instructor: instructor,
		// we don't do any post-filtering, restrict git to only return as many elements as pagination needs.
		MaxWalkDistance: endsAfter,
	}

	err = s.adapter.WalkReferences(ctx, repoPath, handler, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to walk branch references: %w", err)
	}

	log.Ctx(ctx).Trace().Msgf("git adapter returned %d branches", len(branches))

	return branches, nil
}

func listBranchesWalkReferencesHandler(
	branches *[]*types.Branch,
) types.WalkReferencesHandler {
	return func(e types.WalkReferencesEntry) error {
		fullRefName, ok := e[types.GitReferenceFieldRefName]
		if !ok {
			return fmt.Errorf("entry missing reference name")
		}
		objectSHA, ok := e[types.GitReferenceFieldObjectName]
		if !ok {
			return fmt.Errorf("entry missing object sha")
		}

		branch := &types.Branch{
			Name: fullRefName[len(gitReferenceNamePrefixBranch):],
			SHA:  objectSHA,
		}

		// TODO: refactor to not use slice pointers?
		*branches = append(*branches, branch)

		return nil
	}
}
