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

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s RepositoryService) ListTreeNodes(
	request *rpc.ListTreeNodesRequest,
	stream rpc.RepositoryService_ListTreeNodesServer,
) error {
	ctx := stream.Context()
	base := request.GetBase()
	if base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	gitNodes, err := s.adapter.ListTreeNodes(ctx, repoPath,
		request.GetGitRef(), request.GetPath())
	if err != nil {
		return processGitErrorf(err, "failed to list tree nodes")
	}

	log.Ctx(ctx).Trace().Msgf("git adapter returned %d nodes", len(gitNodes))

	for _, gitNode := range gitNodes {
		err = stream.Send(&rpc.ListTreeNodesResponse{
			Node: &rpc.TreeNode{
				Type: mapGitNodeType(gitNode.NodeType),
				Mode: mapGitMode(gitNode.Mode),
				Sha:  gitNode.Sha,
				Name: gitNode.Name,
				Path: gitNode.Path,
			},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send node: %v", err)
		}
	}

	return nil
}

func (s RepositoryService) GetTreeNode(ctx context.Context,
	request *rpc.GetTreeNodeRequest,
) (*rpc.GetTreeNodeResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	gitNode, err := s.adapter.GetTreeNode(ctx, repoPath, request.GitRef, request.Path)
	if err != nil {
		return nil, processGitErrorf(err, "failed to find node '%s' in '%s'", request.Path, request.GitRef)
	}

	res := &rpc.GetTreeNodeResponse{
		Node: &rpc.TreeNode{
			Type: mapGitNodeType(gitNode.NodeType),
			Mode: mapGitMode(gitNode.Mode),
			Sha:  gitNode.Sha,
			Name: gitNode.Name,
			Path: gitNode.Path,
		},
	}

	if !request.GetIncludeLatestCommit() {
		return res, nil
	}

	pathDetails, err := s.adapter.PathsDetails(ctx, repoPath, request.GitRef, []string{request.Path})
	if err != nil {
		return nil, processGitErrorf(err, "failed to get path details for '%s' in '%s'", request.Path, request.GitRef)
	}

	if len(pathDetails) != 1 || pathDetails[0].LastCommit == nil {
		return nil, fmt.Errorf("failed to get details for the path %s", request.Path)
	}

	res.Commit, err = mapGitCommit(pathDetails[0].LastCommit)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s RepositoryService) PathsDetails(ctx context.Context,
	request *rpc.PathsDetailsRequest,
) (*rpc.PathsDetailsResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	pathsDetails, err := s.adapter.PathsDetails(ctx, repoPath, request.GetGitRef(), request.GetPaths())
	if err != nil {
		return nil, processGitErrorf(err, "failed to get path details in '%s'", request.GetGitRef())
	}

	details := make([]*rpc.PathDetails, len(pathsDetails))
	for i, pathDetails := range pathsDetails {
		var lastCommit *rpc.Commit

		if pathDetails.LastCommit != nil {
			lastCommit, err = mapGitCommit(pathDetails.LastCommit)
			if err != nil {
				return nil, fmt.Errorf("failed to map commit: %w", err)
			}
		}

		details[i] = &rpc.PathDetails{
			Path:       pathDetails.Path,
			LastCommit: lastCommit,
			Size:       pathDetails.Size,
		}
	}

	return &rpc.PathsDetailsResponse{
		PathDetails: details,
	}, nil
}
