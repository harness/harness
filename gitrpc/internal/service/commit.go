// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s RepositoryService) GetCommit(ctx context.Context,
	request *rpc.GetCommitRequest) (*rpc.GetCommitResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	// ensure the provided SHA is valid (and not a reference)
	sha := request.GetSha()
	if !isValidGitSHA(sha) {
		return nil, status.Errorf(codes.InvalidArgument, "the provided commit sha '%s' is of invalid format.", sha)
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	gitCommit, err := s.adapter.GetCommit(ctx, repoPath, sha)
	if err != nil {
		return nil, processGitErrorf(err, "failed to get commit")
	}

	commit, err := mapGitCommit(gitCommit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to map git commit: %v", err)
	}

	return &rpc.GetCommitResponse{
		Commit: commit,
	}, nil
}

func (s RepositoryService) ListCommits(request *rpc.ListCommitsRequest,
	stream rpc.RepositoryService_ListCommitsServer) error {
	base := request.GetBase()
	if base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	ctx := stream.Context()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	gitCommits, err := s.adapter.ListCommits(ctx, repoPath, request.GetGitRef(),
		request.GetAfter(), int(request.GetPage()), int(request.GetLimit()))
	if err != nil {
		return processGitErrorf(err, "failed to get list of commits")
	}

	log.Ctx(ctx).Trace().Msgf("git adapter returned %d commits", len(gitCommits))

	for i := range gitCommits {
		var commit *rpc.Commit
		commit, err = mapGitCommit(&gitCommits[i])
		if err != nil {
			return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
		}

		err = stream.Send(&rpc.ListCommitsResponse{
			Commit: commit,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send commit: %v", err)
		}
	}

	return nil
}

func (s RepositoryService) getLatestCommit(ctx context.Context, repoPath string,
	ref string, path string) (*rpc.Commit, error) {
	gitCommit, err := s.adapter.GetLatestCommit(ctx, repoPath, ref, path)
	if err != nil {
		return nil, processGitErrorf(err, "failed to get latest commit")
	}

	return mapGitCommit(gitCommit)
}

func (s RepositoryService) GetCommitDivergences(ctx context.Context,
	request *rpc.GetCommitDivergencesRequest) (*rpc.GetCommitDivergencesResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	// map to gitea requests
	requests := request.GetRequests()
	if requests == nil {
		return nil, status.Error(codes.InvalidArgument, "requests is nil")
	}
	giteaDivergenceRequests := make([]types.CommitDivergenceRequest, len(requests))
	for i := range requests {
		if requests[i] == nil {
			return nil, status.Errorf(codes.InvalidArgument, "requests[%d] is nil", i)
		}
		giteaDivergenceRequests[i].From = requests[i].From
		giteaDivergenceRequests[i].To = requests[i].To
	}

	// call gitea
	giteaDivergenceResponses, err := s.adapter.GetCommitDivergences(ctx, repoPath,
		giteaDivergenceRequests, request.GetMaxCount())
	if err != nil {
		return nil, processGitErrorf(err, "failed to get diverging commits")
	}

	// map to rpc response
	response := &rpc.GetCommitDivergencesResponse{
		Divergences: make([]*rpc.CommitDivergence, len(giteaDivergenceResponses)),
	}
	for i := range giteaDivergenceResponses {
		response.Divergences[i] = &rpc.CommitDivergence{
			Ahead:  giteaDivergenceResponses[i].Ahead,
			Behind: giteaDivergenceResponses[i].Behind,
		}
	}

	return response, nil
}

func (s RepositoryService) MergeBase(ctx context.Context,
	r *rpc.MergeBaseRequest,
) (*rpc.MergeBaseResponse, error) {
	base := r.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	mergeBase, _, err := s.adapter.GetMergeBase(ctx, repoPath, "", r.Ref1, r.Ref2)
	if err != nil {
		return nil, processGitErrorf(err, "failed to find merge base")
	}

	return &rpc.MergeBaseResponse{
		MergeBaseSha: mergeBase,
	}, nil
}
