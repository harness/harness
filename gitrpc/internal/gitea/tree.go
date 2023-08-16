// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/gitrpc/internal/types"

	gitea "code.gitea.io/gitea/modules/git"
	gogitplumbing "github.com/go-git/go-git/v5/plumbing"
	gogitfilemode "github.com/go-git/go-git/v5/plumbing/filemode"
	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
)

func cleanTreePath(treePath string) string {
	return strings.Trim(path.Clean("/"+treePath), "/")
}

// GetTreeNode returns the tree node at the given path as found for the provided reference.
// Note: ref can be Branch / Tag / CommitSHA.
func (g Adapter) GetTreeNode(ctx context.Context,
	repoPath string,
	ref string,
	treePath string,
) (*types.TreeNode, error) {
	treePath = cleanTreePath(treePath)

	repoEntry, err := g.repoCache.Get(ctx, repoPath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to open repository")
	}

	repo := repoEntry.Repo()

	refSHA, err := repo.ResolveRevision(gogitplumbing.Revision(ref))
	if errors.Is(err, gogitplumbing.ErrReferenceNotFound) {
		return nil, types.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to resolve revision %s: %w", ref, err)
	}

	refCommit, err := repo.CommitObject(*refSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to load commit data: %w", err)
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
		if err != nil {
			return nil, fmt.Errorf("can't find path entry %s: %w", treePath, types.ErrPathNotFound)
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
func (g Adapter) ListTreeNodes(ctx context.Context,
	repoPath string,
	ref string,
	treePath string,
) ([]types.TreeNode, error) {
	treePath = cleanTreePath(treePath)

	repoEntry, err := g.repoCache.Get(ctx, repoPath)
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to open repository")
	}

	repo := repoEntry.Repo()

	refSHA, err := repo.ResolveRevision(gogitplumbing.Revision(ref))
	if errors.Is(err, gogitplumbing.ErrReferenceNotFound) {
		return nil, types.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to resolve revision %s: %w", ref, err)
	}

	refCommit, err := repo.CommitObject(*refSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to load commit data: %w", err)
	}

	tree, err := refCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree for the commit: %w", err)
	}

	if len(treePath) > 0 {
		tree, err = tree.Tree(treePath)
		if errors.Is(err, gogitobject.ErrDirectoryNotFound) || errors.Is(err, gogitobject.ErrEntryNotFound) {
			return nil, types.ErrPathNotFound
		} else if err != nil {
			return nil, fmt.Errorf("can't find path entry %s: %w", treePath, err)
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

func (g Adapter) ReadTree(ctx context.Context, repoPath, ref string, w io.Writer, args ...string) error {
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
