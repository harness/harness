// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s RepositoryService) ListTreeNodes(request *rpc.ListTreeNodesRequest,
	stream rpc.RepositoryService_ListTreeNodesServer) error {
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())

	gitNodes, err := s.adapter.ListTreeNodes(stream.Context(), repoPath,
		request.GetGitRef(), request.GetPath(), request.GetRecursive(), request.GetIncludeLatestCommit())
	if err != nil {
		return processGitErrorf(err, "failed to list tree nodes")
	}

	log.Trace().Msgf("git adapter returned %d nodes", len(gitNodes))

	for _, gitNode := range gitNodes {
		var commit *rpc.Commit
		if request.GetIncludeLatestCommit() {
			commit, err = mapGitCommit(gitNode.Commit)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to map git commit: %v", err)
			}
		}

		err = stream.Send(&rpc.ListTreeNodesResponse{
			Node: &rpc.TreeNode{
				Type: mapGitNodeType(gitNode.NodeType),
				Mode: mapGitMode(gitNode.Mode),
				Sha:  gitNode.Sha,
				Name: gitNode.Name,
				Path: gitNode.Path,
			},
			Commit: commit,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to send node: %v", err)
		}
	}

	return nil
}

func (s RepositoryService) GetTreeNode(ctx context.Context,
	request *rpc.GetTreeNodeRequest) (*rpc.GetTreeNodeResponse, error) {
	repoPath := getFullPathForRepo(s.reposRoot, request.GetRepoUid())
	// TODO: do we need to validate request for nil?
	gitNode, err := s.adapter.GetTreeNode(ctx, repoPath, request.GetGitRef(), request.GetPath())
	if err != nil {
		return nil, processGitErrorf(err, "failed to get tree node")
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

	// TODO: improve performance, could be done in lower layer?
	if request.GetIncludeLatestCommit() {
		var commit *rpc.Commit
		commit, err = s.getLatestCommit(ctx, repoPath, request.GetGitRef(), request.GetPath())
		if err != nil {
			return nil, err
		}
		res.Commit = commit
	}

	return res, nil
}
