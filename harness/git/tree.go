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
	SHA  string // TODO: make sha.SHA
	Name string
	Path string
}

type ListTreeNodeParams struct {
	ReadParams
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF             string
	Path               string
	FlattenDirectories bool
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

func (s *Service) GetTreeNode(ctx context.Context, params *GetTreeNodeParams) (*GetTreeNodeOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	gitNode, err := s.git.GetTreeNode(ctx, repoPath, params.GitREF, params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to find node '%s' in '%s': %w", params.Path, params.GitREF, err)
	}

	node, err := mapTreeNode(gitNode)
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc node: %w", err)
	}

	if !params.IncludeLatestCommit {
		return &GetTreeNodeOutput{
			Node: node,
		}, nil
	}

	pathDetails, err := s.git.PathsDetails(ctx, repoPath, params.GitREF, []string{params.Path})
	if err != nil {
		return nil, fmt.Errorf("failed to get path details for '%s' in '%s': %w", params.Path, params.GitREF, err)
	}

	if len(pathDetails) != 1 || pathDetails[0].LastCommit == nil {
		return nil, fmt.Errorf("failed to get details for the path %s", params.Path)
	}
	var commit *Commit
	if pathDetails[0].LastCommit != nil {
		commit, err = mapCommit(pathDetails[0].LastCommit)
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc commit: %w", err)
		}
	}

	return &GetTreeNodeOutput{
		Node:   node,
		Commit: commit,
	}, nil
}

func (s *Service) ListTreeNodes(ctx context.Context, params *ListTreeNodeParams) (*ListTreeNodeOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	res, err := s.git.ListTreeNodes(ctx, repoPath, params.GitREF, params.Path, params.FlattenDirectories)
	if err != nil {
		return nil, fmt.Errorf("failed to list tree nodes: %w", err)
	}

	nodes := make([]TreeNode, len(res))
	for i := range res {
		n, err := mapTreeNode(&res[i])
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc node: %w", err)
		}

		nodes[i] = n
	}

	return &ListTreeNodeOutput{
		Nodes: nodes,
	}, nil
}

type ListPathsParams struct {
	ReadParams
	// GitREF is a git reference (branch / tag / commit SHA)
	GitREF             string
	IncludeDirectories bool
}

type ListPathsOutput struct {
	Files       []string
	Directories []string
}

func (s *Service) ListPaths(ctx context.Context, params *ListPathsParams) (*ListPathsOutput, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	files, dirs, err := s.git.ListPaths(
		ctx,
		repoPath,
		params.GitREF,
		params.IncludeDirectories,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list paths: %w", err)
	}

	return &ListPathsOutput{
			Files:       files,
			Directories: dirs,
		},
		nil
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
}

func (s *Service) PathsDetails(ctx context.Context, params PathsDetailsParams) (PathsDetailsOutput, error) {
	if err := params.Validate(); err != nil {
		return PathsDetailsOutput{}, err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	pathsDetails, err := s.git.PathsDetails(
		ctx,
		repoPath,
		params.GitREF,
		params.Paths)
	if err != nil {
		return PathsDetailsOutput{}, fmt.Errorf("failed to get path details in '%s': %w", params.GitREF, err)
	}

	details := make([]PathDetails, len(pathsDetails))
	for i, pathDetail := range pathsDetails {
		var lastCommit *Commit

		if pathDetail.LastCommit != nil {
			lastCommit, err = mapCommit(pathDetail.LastCommit)
			if err != nil {
				return PathsDetailsOutput{}, fmt.Errorf("failed to map last commit: %w", err)
			}
		}

		details[i] = PathDetails{
			Path:       pathDetail.Path,
			LastCommit: lastCommit,
		}
	}

	return PathsDetailsOutput{
		Details: details,
	}, nil
}
