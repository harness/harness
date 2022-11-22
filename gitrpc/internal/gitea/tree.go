// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/gitrpc/internal/types"

	gitea "code.gitea.io/gitea/modules/git"
)

func cleanTreePath(treePath string) string {
	return strings.Trim(path.Clean("/"+treePath), "/")
}

// GetTreeNode returns the tree node at the given path as found for the provided reference.
// Note: ref can be Branch / Tag / CommitSHA.
func (g Adapter) GetTreeNode(ctx context.Context, repoPath string,
	ref string, treePath string) (*types.TreeNode, error) {
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
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
// IMPORTANT: recursive and includeLatestCommit can't be used together.
// Note: ref can be Branch / Tag / CommitSHA.
//
//nolint:gocognit // refactor if needed
func (g Adapter) ListTreeNodes(ctx context.Context, repoPath string,
	ref string, treePath string, recursive bool, includeLatestCommit bool) ([]types.TreeNodeWithCommit, error) {
	if recursive && includeLatestCommit {
		// To avoid potential performance catastrophies, block recursive with includeLatestCommit
		// TODO: this should return bad error to caller if needed?
		// TODO: should this be refactored in two methods?
		return nil, fmt.Errorf("latest commit with recursive query is not supported")
	}

	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
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

	var giteaEntries gitea.Entries
	if recursive {
		giteaEntries, err = giteaTree.ListEntriesRecursive()
	} else {
		giteaEntries, err = giteaTree.ListEntries()
	}
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to list entries for tree '%s'", treePath)
	}

	var latestCommits []gitea.CommitInfo
	if includeLatestCommit {
		// TODO: can be speed up with latestCommitCache (currently nil)
		latestCommits, _, err = giteaEntries.GetCommitsInfo(ctx, giteaCommit, treePath, nil)
		if err != nil {
			return nil, processGiteaErrorf(err, "failed to get latest commits for entries")
		}

		if len(latestCommits) != len(giteaEntries) {
			return nil, fmt.Errorf("latest commit info doesn't match tree node info - count differs")
		}
	}

	nodes := make([]types.TreeNodeWithCommit, len(giteaEntries))
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

		var commit *types.Commit
		if includeLatestCommit {
			commit, err = mapGiteaCommit(latestCommits[i].Commit)
			if err != nil {
				return nil, err
			}
		}

		nodes[i] = types.TreeNodeWithCommit{
			TreeNode: types.TreeNode{
				NodeType: nodeType,
				Mode:     mode,
				Sha:      giteaEntry.ID.String(),
				Name:     name,
				Path:     filepath.Join(treePath, relPath),
			},
			Commit: commit,
		}
	}

	return nodes, nil
}
