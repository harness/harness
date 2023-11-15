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

package adapter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	gogitfilemode "github.com/go-git/go-git/v5/plumbing/filemode"
	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
)

func cleanTreePath(treePath string) string {
	return strings.Trim(path.Clean("/"+treePath), "/")
}

// GetTreeNode returns the tree node at the given path as found for the provided reference.
// Note: ref can be Branch / Tag / CommitSHA.
func (a Adapter) GetTreeNode(
	ctx context.Context,
	repoPath string,
	ref string,
	treePath string,
) (*types.TreeNode, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	treePath = cleanTreePath(treePath)

	_, refCommit, err := a.getGoGitCommit(ctx, repoPath, ref)
	if err != nil {
		return nil, err
	}

	rootEntry := gogitobject.TreeEntry{
		Name: "",
		Mode: gogitfilemode.Dir,
		Hash: refCommit.TreeHash,
	}

	treeEntry := &rootEntry

	if len(treePath) > 0 {
		tree, err := refCommit.Tree()
		if err != nil {
			return nil, fmt.Errorf("failed to get tree for the commit: %w", err)
		}

		treeEntry, err = tree.FindEntry(treePath)
		if errors.Is(err, gogitobject.ErrDirectoryNotFound) || errors.Is(err, gogitobject.ErrEntryNotFound) {
			return nil, errors.NotFound("path not found '%s'", treePath)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to find path entry %s: %w", treePath, err)
		}
	}

	nodeType, mode, err := mapGogitNodeToTreeNodeModeAndType(treeEntry.Mode)
	if err != nil {
		return nil, err
	}

	return &types.TreeNode{
		Mode:     mode,
		NodeType: nodeType,
		Sha:      treeEntry.Hash.String(),
		Name:     treeEntry.Name,
		Path:     treePath,
	}, nil
}

// ListTreeNodes lists the child nodes of a tree reachable from ref via the specified path
// and includes the latest commit for all nodes if requested.
// Note: ref can be Branch / Tag / CommitSHA.
//
//nolint:gocognit // refactor if needed
func (a Adapter) ListTreeNodes(
	ctx context.Context,
	repoPath string,
	ref string,
	treePath string,
) ([]types.TreeNode, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	treePath = cleanTreePath(treePath)

	_, refCommit, err := a.getGoGitCommit(ctx, repoPath, ref)
	if err != nil {
		return nil, err
	}

	tree, err := refCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for the commit: %w", err)
	}

	if len(treePath) > 0 {
		tree, err = tree.Tree(treePath)
		if errors.Is(err, gogitobject.ErrDirectoryNotFound) || errors.Is(err, gogitobject.ErrEntryNotFound) {
			return nil, &types.PathNotFoundError{Path: treePath}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to find path entry %s: %w", treePath, err)
		}
	}

	treeNodes := make([]types.TreeNode, len(tree.Entries))
	for i, treeEntry := range tree.Entries {
		nodeType, mode, err := mapGogitNodeToTreeNodeModeAndType(treeEntry.Mode)
		if err != nil {
			return nil, err
		}

		treeNodes[i] = types.TreeNode{
			NodeType: nodeType,
			Mode:     mode,
			Sha:      treeEntry.Hash.String(),
			Name:     treeEntry.Name,
			Path:     filepath.Join(treePath, treeEntry.Name),
		}
	}

	return treeNodes, nil
}

func (a Adapter) ReadTree(
	ctx context.Context,
	repoPath string,
	ref string,
	w io.Writer,
	args ...string,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	errbuf := bytes.Buffer{}
	if err := gitea.NewCommand(ctx, append([]string{"read-tree", ref}, args...)...).
		Run(&gitea.RunOpts{
			Dir:    repoPath,
			Stdout: w,
			Stderr: &errbuf,
		}); err != nil {
		return fmt.Errorf("unable to read %s in to the index: %w\n%s",
			ref, err, errbuf.String())
	}
	return nil
}
