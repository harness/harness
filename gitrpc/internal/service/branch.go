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
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/gitrpc/check"
	"github.com/harness/gitness/gitrpc/internal/gitea"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var listBranchesRefFields = []types.GitReferenceField{types.GitReferenceFieldRefName, types.GitReferenceFieldObjectName}

func (s ReferenceService) CreateBranch(
	ctx context.Context,
	request *rpc.CreateBranchRequest,
) (*rpc.CreateBranchResponse, error) {
	if err := check.BranchName(request.BranchName); err != nil {
		return nil, ErrInvalidArgument(err)
	}

	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	if ok, err := repoIsEmpty(ctx, repoPath); ok {
		return nil, ErrInvalidArgumentf("branch cannot be created on empty repository", err)
	}

	sharedRepo, err := NewSharedRepo(s.tmpDir, base.GetRepoUid(), repoPath)
	if err != nil {
		return nil, processGitErrorf(err, "failed to create new shared repo")
	}
	defer sharedRepo.Close(ctx)

	// clone repo (with HEAD branch - target might be anything)
	err = sharedRepo.Clone(ctx, "")
	if err != nil {
		return nil, processGitErrorf(err, "failed to clone shared repo with branch '%s'", request.GetBranchName())
	}

	_, err = sharedRepo.GetBranchCommit(request.GetBranchName())
	// return an error if branch alredy exists (push doesn't fail if it's a noop or fast forward push)
	if err == nil {
		return nil, ErrAlreadyExistsf("branch '%s' already exists", request.GetBranchName())
	}
	if !git.IsErrNotExist(err) {
		return nil, processGitErrorf(err, "branch creation of '%s' failed", request.GetBranchName())
	}

	// get target commit (as target could be branch/tag/commit, and tag can't be pushed using source:destination syntax)
	targetCommit, err := s.adapter.GetCommit(ctx, sharedRepo.tmpPath, strings.TrimSpace(request.GetTarget()))
	if git.IsErrNotExist(err) {
		return nil, ErrNotFoundf("target '%s' doesn't exist", request.GetTarget())
	}
	if err != nil {
		return nil, processGitErrorf(err, "failed to get commit id for target '%s'", request.GetTarget())
	}

	// push to new branch (all changes should go through push flow for hooks and other safety meassures)
	err = sharedRepo.PushCommitToBranch(ctx, base, targetCommit.SHA, request.GetBranchName())
	if err != nil {
		return nil, processGitErrorf(err, "failed to push new branch '%s'", request.GetBranchName())
	}

	// get branch
	// TODO: get it from shared repo to avoid opening another gitea repo and having to strip here.
	gitBranch, err := s.adapter.GetBranch(ctx, repoPath,
		strings.TrimPrefix(request.GetBranchName(), gitReferenceNamePrefixBranch))
	if err != nil {
		return nil, processGitErrorf(err, "failed to get gitea branch '%s'", request.GetBranchName())
	}

	branch, err := mapGitBranch(gitBranch)
	if err != nil {
		return nil, err
	}

	return &rpc.CreateBranchResponse{
		Branch: branch,
	}, nil
}

func (s ReferenceService) GetBranch(ctx context.Context,
	request *rpc.GetBranchRequest) (*rpc.GetBranchResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	gitBranch, err := s.adapter.GetBranch(ctx, repoPath,
		strings.TrimPrefix(request.GetBranchName(), gitReferenceNamePrefixBranch))
	if err != nil {
		return nil, processGitErrorf(err, "failed to get gitea branch '%s'", request.GetBranchName())
	}

	branch, err := mapGitBranch(gitBranch)
	if err != nil {
		return nil, err
	}

	return &rpc.GetBranchResponse{
		Branch: branch,
	}, nil
}

func (s ReferenceService) DeleteBranch(ctx context.Context,
	request *rpc.DeleteBranchRequest) (*rpc.DeleteBranchResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	sharedRepo, err := NewSharedRepo(s.tmpDir, base.GetRepoUid(), repoPath)
	if err != nil {
		return nil, processGitErrorf(err, "failed to create new shared repo")
	}
	defer sharedRepo.Close(ctx)

	// clone repo (technically we don't care about which branch we clone)
	err = sharedRepo.Clone(ctx, request.GetBranchName())
	if err != nil {
		return nil, processGitErrorf(err, "failed to clone shared repo with branch '%s'", request.GetBranchName())
	}

	// get latest branch commit before we delete
	gitCommit, err := sharedRepo.GetBranchCommit(request.GetBranchName())
	if err != nil {
		return nil, processGitErrorf(err, "failed to get gitea commit for branch '%s'", request.GetBranchName())
	}

	// push to new branch (all changes should go through push flow for hooks and other safety meassures)
	// NOTE: setting sourceRef to empty will delete the remote branch when pushing:
	// https://git-scm.com/docs/git-push#Documentation/git-push.txt-ltrefspecgt82308203
	err = sharedRepo.PushDeleteBranch(ctx, base, request.GetBranchName())
	if err != nil {
		return nil, processGitErrorf(err, "failed to delete branch '%s' from remote repo", request.GetBranchName())
	}

	return &rpc.DeleteBranchResponse{
		Sha: gitCommit.ID.String(),
	}, nil
}

func (s ReferenceService) ListBranches(request *rpc.ListBranchesRequest,
	stream rpc.ReferenceService_ListBranchesServer) error {
	base := request.GetBase()
	if base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	ctx := stream.Context()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	// get all required information from git references
	branches, err := s.listBranchesLoadReferenceData(ctx, repoPath, request)
	if err != nil {
		return err
	}

	// get commits if needed (single call for perf savings: 1s-4s vs 5s-20s)
	if request.GetIncludeCommit() {
		commitSHAs := make([]string, len(branches))
		for i := range branches {
			commitSHAs[i] = branches[i].Sha
		}

		var gitCommits []types.Commit
		gitCommits, err = s.adapter.GetCommits(ctx, repoPath, commitSHAs)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get commits: %v", err)
		}

		for i := range gitCommits {
			branches[i].Commit, err = mapGitCommit(&gitCommits[i])
			if err != nil {
				return err
			}
		}
	}

	// send out all branches
	for _, branch := range branches {
		err = stream.Send(&rpc.ListBranchesResponse{
			Branch: branch,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send branch: %v", err)
		}
	}

	return nil
}

func (s ReferenceService) listBranchesLoadReferenceData(ctx context.Context,
	repoPath string, request *rpc.ListBranchesRequest) ([]*rpc.Branch, error) {
	// TODO: can we be smarter with slice allocation
	branches := make([]*rpc.Branch, 0, 16)
	handler := listBranchesWalkReferencesHandler(&branches)
	instructor, endsAfter, err := wrapInstructorWithOptionalPagination(
		gitea.DefaultInstructor, // branches only have one target type, default instructor is enough
		request.GetPage(),
		request.GetPageSize())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid pagination details: %v", err)
	}

	opts := &types.WalkReferencesOptions{
		Patterns:   createReferenceWalkPatternsFromQuery(gitReferenceNamePrefixBranch, request.GetQuery()),
		Sort:       mapListBranchesSortOption(request.Sort),
		Order:      mapSortOrder(request.Order),
		Fields:     listBranchesRefFields,
		Instructor: instructor,
		// we don't do any post-filtering, restrict git to only return as many elements as pagination needs.
		MaxWalkDistance: endsAfter,
	}

	err = s.adapter.WalkReferences(ctx, repoPath, handler, opts)
	if err != nil {
		return nil, processGitErrorf(err, "failed to walk branch references")
	}

	log.Ctx(ctx).Trace().Msgf("git adapter returned %d branches", len(branches))

	return branches, nil
}

func listBranchesWalkReferencesHandler(branches *[]*rpc.Branch) types.WalkReferencesHandler {
	return func(e types.WalkReferencesEntry) error {
		fullRefName, ok := e[types.GitReferenceFieldRefName]
		if !ok {
			return fmt.Errorf("entry missing reference name")
		}
		objectSHA, ok := e[types.GitReferenceFieldObjectName]
		if !ok {
			return fmt.Errorf("entry missing object sha")
		}

		branch := &rpc.Branch{
			Name: fullRefName[len(gitReferenceNamePrefixBranch):],
			Sha:  objectSHA,
		}

		// TODO: refactor to not use slice pointers?
		*branches = append(*branches, branch)

		return nil
	}
}
