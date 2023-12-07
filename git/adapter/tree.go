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

	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
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
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to open repository")
	}
	defer giteaRepo.Close()

	// Get the giteaCommit object for the ref
	giteaCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, processGiteaErrorf(err, "error getting commit for ref '%s'", ref)
	}

	// TODO: handle ErrNotExist :)
	giteaTreeEntry, err := giteaCommit.GetTreeEntryByPath(treePath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to get tree entry for commit '%s' at path '%s'",
			giteaCommit.ID.String(), treePath)
	}

	nodeType, mode, err := mapGiteaNodeToTreeNodeModeAndType(giteaTreeEntry.Mode())
	if err != nil {
		return nil, err
	}

	return &types.TreeNode{
		Mode:     mode,
		NodeType: nodeType,
		Sha:      giteaTreeEntry.ID.String(),
		Name:     giteaTreeEntry.Name(),
		Path:     treePath,
	}, nil
}

// ListTreeNodes lists the child nodes of a tree reachable from ref via the specified path
// and includes the latest commit for all nodes if requested.
// Note: ref can be Branch / Tag / CommitSHA.
func (a Adapter) ListTreeNodes(
	ctx context.Context,
	repoPath string,
	ref string,
	treePath string,
) ([]types.TreeNode, error) {
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to open repository")
	}
	defer giteaRepo.Close()

	// Get the giteaCommit object for the ref
	giteaCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, processGiteaErrorf(err, "error getting commit for ref '%s'", ref)
	}

	// Get the giteaTree object for the ref
	giteaTree, err := giteaCommit.SubTree(treePath)
	if err != nil {
		return nil, processGiteaErrorf(err, "error getting tree for '%s'", treePath)
	}

	giteaEntries, err := giteaTree.ListEntries()
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to list entries for tree '%s'", treePath)
	}

	nodes := make([]types.TreeNode, len(giteaEntries))
	for i := range giteaEntries {
		giteaEntry := giteaEntries[i]

		var nodeType types.TreeNodeType
		var mode types.TreeNodeMode
		nodeType, mode, err = mapGiteaNodeToTreeNodeModeAndType(giteaEntry.Mode())
		if err != nil {
			return nil, err
		}

		// giteaNode.Name() returns the path of the node relative to the tree.
		relPath := giteaEntry.Name()
		name := filepath.Base(relPath)

		nodes[i] = types.TreeNode{
			NodeType: nodeType,
			Mode:     mode,
			Sha:      giteaEntry.ID.String(),
			Name:     name,
			Path:     filepath.Join(treePath, relPath),
		}
	}

	return nodes, nil
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
