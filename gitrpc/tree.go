// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
	// RepoUID is the uid of the git repository
	RepoUID string
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF              string
	Path                string
	IncludeLatestCommit bool
	Recursive           bool
}

type ListTreeNodeOutput struct {
	Nodes []TreeNodeWithCommit
}

type TreeNodeWithCommit struct {
	TreeNode
	Commit *Commit
}

type GetTreeNodeParams struct {
	// RepoUID is the uid of the git repository
	RepoUID string
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
		RepoUid:             params.RepoUID,
		GitRef:              params.GitREF,
		Path:                params.Path,
		IncludeLatestCommit: params.IncludeLatestCommit,
	})
	if err != nil {
		return nil, err
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
		RepoUid:             params.RepoUID,
		GitRef:              params.GitREF,
		Path:                params.Path,
		IncludeLatestCommit: params.IncludeLatestCommit,
		Recursive:           params.Recursive,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for tree nodes: %w", err)
	}

	nodes := make([]TreeNodeWithCommit, 0, 16)
	for {
		var next *rpc.ListTreeNodesResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, fmt.Errorf("received unexpected error from rpc: %w", err)
		}

		var node TreeNode
		node, err = mapRPCTreeNode(next.GetNode())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc node: %w", err)
		}
		var commit *Commit
		if next.GetCommit() != nil {
			commit, err = mapRPCCommit(next.GetCommit())
			if err != nil {
				return nil, fmt.Errorf("failed to map rpc commit: %w", err)
			}
		}

		nodes = append(nodes, TreeNodeWithCommit{
			TreeNode: node,
			Commit:   commit,
		})
	}

	return &ListTreeNodeOutput{
		Nodes: nodes,
	}, nil
}
