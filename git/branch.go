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
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/check"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"

	"github.com/rs/zerolog/log"
)

type BranchSortOption int

const (
	BranchSortOptionDefault BranchSortOption = iota
	BranchSortOptionName
	BranchSortOptionDate
)

var listBranchesRefFields = []api.GitReferenceField{
	api.GitReferenceFieldRefName,
	api.GitReferenceFieldObjectName,
}

type Branch struct {
	Name   string
	SHA    sha.SHA
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
	// BranchName is the name of the branch
	BranchName string
	SHA        string
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
	if err := check.BranchName(params.BranchName); err != nil {
		return nil, errors.InvalidArgument(err.Error())
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	targetCommit, err := s.git.GetCommit(ctx, repoPath, strings.TrimSpace(params.Target))
	if err != nil {
		return nil, fmt.Errorf("failed to get target commit: %w", err)
	}

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create ref updater to create the branch: %w", err)
	}

	branchRef := api.GetReferenceFromBranchName(params.BranchName)

	err = refUpdater.DoOne(ctx, branchRef, sha.Nil, targetCommit.SHA)
	if errors.IsConflict(err) {
		return nil, errors.Conflict("branch %q already exists", params.BranchName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create branch reference: %w", err)
	}

	commit, err := mapCommit(targetCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to map git commit: %w", err)
	}

	return &CreateBranchOutput{
		Branch: Branch{
			Name:   params.BranchName,
			SHA:    commit.SHA,
			Commit: commit,
		},
	}, nil
}

func (s *Service) GetBranch(ctx context.Context, params *GetBranchParams) (*GetBranchOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	sanitizedBranchName := strings.TrimPrefix(params.BranchName, gitReferenceNamePrefixBranch)

	gitBranch, err := s.git.GetBranch(ctx, repoPath, sanitizedBranchName)
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
	commitSha, _ := sha.NewOrEmpty(params.SHA)

	refUpdater, err := hook.CreateRefUpdater(s.hookClientFactory, params.EnvVars, repoPath)
	if err != nil {
		return fmt.Errorf("failed to create ref updater to create the branch: %w", err)
	}

	branchRef := api.GetReferenceFromBranchName(params.BranchName)

	err = refUpdater.DoOne(ctx, branchRef, commitSha, sha.Nil)
	if errors.IsNotFound(err) {
		return errors.NotFound("branch %q does not exist", params.BranchName)
	}
	if err != nil {
		return fmt.Errorf("failed to delete branch reference: %w", err)
	}

	return nil
}

func (s *Service) ListBranches(ctx context.Context, params *ListBranchesParams) (*ListBranchesOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	gitBranches, err := s.listBranchesLoadReferenceData(ctx, repoPath, api.BranchFilter{
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
			commitSHAs[i] = gitBranches[i].SHA.String()
		}

		var gitCommits []*api.Commit
		gitCommits, err = s.git.GetCommits(ctx, repoPath, commitSHAs)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit: %w", err)
		}

		for i := range gitCommits {
			gitBranches[i].Commit = gitCommits[i]
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
	filter api.BranchFilter,
) ([]*api.Branch, error) {
	// TODO: can we be smarter with slice allocation
	branches := make([]*api.Branch, 0, 16)
	handler := listBranchesWalkReferencesHandler(&branches)
	instructor, endsAfter, err := wrapInstructorWithOptionalPagination(
		api.DefaultInstructor, // branches only have one target type, default instructor is enough
		filter.Page,
		filter.PageSize,
	)
	if err != nil {
		return nil, errors.InvalidArgument("invalid pagination details: %v", err)
	}

	opts := &api.WalkReferencesOptions{
		Patterns:   createReferenceWalkPatternsFromQuery(gitReferenceNamePrefixBranch, filter.Query),
		Sort:       filter.Sort,
		Order:      filter.Order,
		Fields:     listBranchesRefFields,
		Instructor: instructor,
		// we don't do any post-filtering, restrict git to only return as many elements as pagination needs.
		MaxWalkDistance: endsAfter,
	}

	err = s.git.WalkReferences(ctx, repoPath, handler, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to walk branch references: %w", err)
	}

	log.Ctx(ctx).Trace().Msgf("git api returned %d branches", len(branches))

	return branches, nil
}

func listBranchesWalkReferencesHandler(
	branches *[]*api.Branch,
) api.WalkReferencesHandler {
	return func(e api.WalkReferencesEntry) error {
		fullRefName, ok := e[api.GitReferenceFieldRefName]
		if !ok {
			return fmt.Errorf("entry missing reference name")
		}
		objectSHA, ok := e[api.GitReferenceFieldObjectName]
		if !ok {
			return fmt.Errorf("entry missing object sha")
		}

		branch := &api.Branch{
			Name: fullRefName[len(gitReferenceNamePrefixBranch):],
			SHA:  sha.Must(objectSHA),
		}

		// TODO: refactor to not use slice pointers?
		*branches = append(*branches, branch)

		return nil
	}
}
