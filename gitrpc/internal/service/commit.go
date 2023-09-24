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
	"strconv"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

	gitCommits, renameDetails, err := s.adapter.ListCommits(ctx, repoPath, request.GetGitRef(),
		int(request.GetPage()), int(request.GetLimit()), types.CommitFilter{AfterRef: request.After,
			Path:      request.Path,
			Since:     request.Since,
			Until:     request.Until,
			Committer: request.Committer})
	if err != nil {
		return processGitErrorf(err, "failed to get list of commits")
	}

	// try to get total commits between gitref and After refs
	totalCommits := 0
	if request.Page == 1 && len(gitCommits) < int(request.Limit) {
		totalCommits = len(gitCommits)
	} else if request.After != "" && request.GitRef != request.After {
		div, err := s.adapter.GetCommitDivergences(ctx, repoPath, []types.CommitDivergenceRequest{
			{From: request.GitRef, To: request.After},
		}, 0)
		if err != nil {
			return processGitErrorf(err, "failed to get total commits")
		}
		if len(div) > 0 {
			totalCommits = int(div[0].Ahead)
		}
	}

	log.Ctx(ctx).Trace().Msgf("git adapter returned %d commits", len(gitCommits))
	header := metadata.New(map[string]string{"total-commits": strconv.Itoa(totalCommits)})
	if err := stream.SendHeader(header); err != nil {
		return ErrInternalf("unable to send 'total-commits' header", err)
	}

	for i := range gitCommits {
		var commit *rpc.Commit
		commit, err = mapGitCommit(&gitCommits[i])
		if err != nil {
			return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
		}

		err = stream.Send(&rpc.ListCommitsResponse{
			Commit:        commit,
			RenameDetails: mapRenameDetails(renameDetails),
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send commit: %v", err)
		}
	}

	return nil
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
