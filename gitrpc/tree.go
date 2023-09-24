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

package gitrpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
)

// TreeNodeType specifies the different types of nodes in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeType (proto).
type TreeNodeType string

const (
	TreeNodeTypeTree   TreeNodeType = "tree"
	TreeNodeTypeBlob   TreeNodeType = "blob"
	TreeNodeTypeCommit TreeNodeType = "commit"
)

// TreeNodeMode specifies the different modes of a node in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeMode (proto).
type TreeNodeMode string

const (
	TreeNodeModeFile    TreeNodeMode = "file"
	TreeNodeModeSymlink TreeNodeMode = "symlink"
	TreeNodeModeExec    TreeNodeMode = "exec"
	TreeNodeModeTree    TreeNodeMode = "tree"
	TreeNodeModeCommit  TreeNodeMode = "commit"
)

type TreeNode struct {
	Type TreeNodeType
	Mode TreeNodeMode
	SHA  string
	Name string
	Path string
}

type ListTreeNodeParams struct {
	ReadParams
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF              string
	Path                string
	IncludeLatestCommit bool
}

type ListTreeNodeOutput struct {
	Nodes []TreeNode
}

type GetTreeNodeParams struct {
	ReadParams
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF              string
	Path                string
	IncludeLatestCommit bool
}

type GetTreeNodeOutput struct {
	Node   TreeNode
	Commit *Commit
}

func (c *Client) GetTreeNode(ctx context.Context, params *GetTreeNodeParams) (*GetTreeNodeOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	resp, err := c.repoService.GetTreeNode(ctx, &rpc.GetTreeNodeRequest{
		Base:                mapToRPCReadRequest(params.ReadParams),
		GitRef:              params.GitREF,
		Path:                params.Path,
		IncludeLatestCommit: params.IncludeLatestCommit,
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to get tree node from server")
	}

	node, err := mapRPCTreeNode(resp.GetNode())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc node: %w", err)
	}

	var commit *Commit
	if resp.GetCommit() != nil {
		commit, err = mapRPCCommit(resp.GetCommit())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc commit: %w", err)
		}
	}

	return &GetTreeNodeOutput{
		Node:   node,
		Commit: commit,
	}, nil
}

func (c *Client) ListTreeNodes(ctx context.Context, params *ListTreeNodeParams) (*ListTreeNodeOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	stream, err := c.repoService.ListTreeNodes(ctx, &rpc.ListTreeNodesRequest{
		Base:   mapToRPCReadRequest(params.ReadParams),
		GitRef: params.GitREF,
		Path:   params.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for tree nodes: %w", err)
	}

	nodes := make([]TreeNode, 0, 16)
	for {
		var next *rpc.ListTreeNodesResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, processRPCErrorf(err, "received unexpected error from server")
		}

		var node TreeNode
		node, err = mapRPCTreeNode(next.GetNode())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc node: %w", err)
		}

		nodes = append(nodes, node)
	}

	return &ListTreeNodeOutput{
		Nodes: nodes,
	}, nil
}

type PathsDetailsParams struct {
	ReadParams
	GitREF string
	Paths  []string
}

type PathsDetailsOutput struct {
	Details []PathDetails
}

type PathDetails struct {
	Path       string  `json:"path"`
	LastCommit *Commit `json:"last_commit,omitempty"`
	Size       int64   `json:"size,omitempty"`
}

func (c *Client) PathsDetails(ctx context.Context, params PathsDetailsParams) (PathsDetailsOutput, error) {
	response, err := c.repoService.PathsDetails(ctx, &rpc.PathsDetailsRequest{
		Base:   mapToRPCReadRequest(params.ReadParams),
		GitRef: params.GitREF,
		Paths:  params.Paths,
	})
	if err != nil {
		return PathsDetailsOutput{}, processRPCErrorf(err, "failed to get paths details")
	}

	details := make([]PathDetails, len(response.PathDetails))
	for i, pathDetail := range response.PathDetails {
		var lastCommit *Commit

		if pathDetail.LastCommit != nil {
			lastCommit, err = mapRPCCommit(pathDetail.LastCommit)
			if err != nil {
				return PathsDetailsOutput{}, fmt.Errorf("failed to map last commit: %w", err)
			}
		}

		details[i] = PathDetails{
			Path:       pathDetail.Path,
			Size:       pathDetail.Size,
			LastCommit: lastCommit,
		}
	}

	return PathsDetailsOutput{
		Details: details,
	}, nil
}
