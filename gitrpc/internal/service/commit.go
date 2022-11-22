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

func (s RepositoryService) ListCommits(request *rpc.ListCommitsRequest,
	stream rpc.RepositoryService_ListCommitsServer) error {
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())

	gitCommits, totalCount, err := s.adapter.ListCommits(stream.Context(), repoPath, request.GetGitRef(),
		int(request.GetPage()), int(request.GetPageSize()))
	if err != nil {
		return processGitErrorf(err, "failed to get list of commits")
	}

	log.Trace().Msgf("git adapter returned %d commits (total: %d)", len(gitCommits), totalCount)

	// send info about total number of commits first
	err = stream.Send(&rpc.ListCommitsResponse{
		Data: &rpc.ListCommitsResponse_Header{
			Header: &rpc.ListCommitsResponseHeader{
				TotalCount: totalCount,
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send response header: %v", err)
	}

	for i := range gitCommits {
		var commit *rpc.Commit
		commit, err = mapGitCommit(&gitCommits[i])
		if err != nil {
			return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
		}

		err = stream.Send(&rpc.ListCommitsResponse{
			Data: &rpc.ListCommitsResponse_Commit{
				Commit: commit,
			},
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
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())

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
