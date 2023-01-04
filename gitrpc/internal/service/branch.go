// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc/events"
	"github.com/harness/gitness/gitrpc/internal/gitea"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var listBranchesRefFields = []types.GitReferenceField{types.GitReferenceFieldRefName, types.GitReferenceFieldObjectName}

func (s ReferenceService) CreateBranch(ctx context.Context,
	request *rpc.CreateBranchRequest) (*rpc.CreateBranchResponse, error) {
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())

	gitBranch, err := s.adapter.CreateBranch(ctx, repoPath, request.GetBranchName(), request.GetTarget())
	if err != nil {
		return nil, processGitErrorf(err, "failed to create branch")
	}

	// at this point the branch got created (emit event even if we'd fail to map the git branch)
	s.eventReporter.BranchCreated(ctx, &events.BranchCreatedPayload{
		RepoUID:    request.RepoUid,
		BranchName: request.BranchName,
		FullRef:    fmt.Sprintf("refs/heads/%s", request.BranchName),
		SHA:        gitBranch.SHA,
	})

	branch, err := mapGitBranch(gitBranch)
	if err != nil {
		return nil, err
	}

	return &rpc.CreateBranchResponse{
		Branch: branch,
	}, nil
}

func (s ReferenceService) DeleteBranch(ctx context.Context,
	request *rpc.DeleteBranchRequest) (*rpc.DeleteBranchResponse, error) {
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())

	// TODO: block deletion of protected branch (in the future)
	sha, err := s.adapter.DeleteBranch(ctx, repoPath, request.GetBranchName(), request.GetForce())
	if err != nil {
		return nil, processGitErrorf(err, "failed to delete branch")
	}

	// at this point the branch got created (emit event even if we'd fail to map the git branch)
	s.eventReporter.BranchDeleted(ctx, &events.BranchDeletedPayload{
		RepoUID:    request.RepoUid,
		BranchName: request.BranchName,
		FullRef:    fmt.Sprintf("refs/heads/%s", request.BranchName),
		SHA:        sha,
	})

	return &rpc.DeleteBranchResponse{}, nil
}

func (s ReferenceService) ListBranches(request *rpc.ListBranchesRequest,
	stream rpc.ReferenceService_ListBranchesServer) error {
	ctx := stream.Context()
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())

	// get all required information from git refrences
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
