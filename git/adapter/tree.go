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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
)

func cleanTreePath(treePath string) string {
	return strings.Trim(path.Clean("/"+treePath), "/")
}

func parseTreeNodeMode(s string) (types.TreeNodeType, types.TreeNodeMode, error) {
	switch s {
	case "100644":
		return types.TreeNodeTypeBlob, types.TreeNodeModeFile, nil
	case "120000":
		return types.TreeNodeTypeBlob, types.TreeNodeModeSymlink, nil
	case "100755":
		return types.TreeNodeTypeBlob, types.TreeNodeModeExec, nil
	case "160000":
		return types.TreeNodeTypeCommit, types.TreeNodeModeCommit, nil
	case "040000":
		return types.TreeNodeTypeTree, types.TreeNodeModeTree, nil
	default:
		return types.TreeNodeTypeBlob, types.TreeNodeModeFile,
			fmt.Errorf("unknown git tree node mode: '%s'", s)
	}
}

func scanZeroSeparated(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil // Return nothing if at end of file and no data passed
	}
	if i := strings.IndexByte(string(data), 0); i >= 0 {
		return i + 1, data[0:i], nil // Split at zero byte
	}
	if atEOF {
		return len(data), data, nil // at the end of file return the data
	}
	return
}

var regexpLsTreeColumns = regexp.MustCompile(`^(\d{6})\s+(\w+)\s+(\w+)\t(.+)$`)

func lsTree(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
) ([]types.TreeNode, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}

	args := []string{"ls-tree", "-z", rev, treePath}
	output, stderr, err := gitea.NewCommand(ctx, args...).RunStdString(&gitea.RunOpts{Dir: repoPath})
	if strings.Contains(stderr, "fatal: Not a valid object name") {
		return nil, errors.NotFound("revision %q not found", rev)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to run git ls-tree: %w", err)
	}

	if output == "" {
		return nil, &types.PathNotFoundError{Path: treePath}
	}

	n := strings.Count(output, "\x00")

	list := make([]types.TreeNode, 0, n)
	scan := bufio.NewScanner(strings.NewReader(output))
	scan.Split(scanZeroSeparated)
	for scan.Scan() {
		columns := regexpLsTreeColumns.FindStringSubmatch(scan.Text())
		if columns == nil {
			return nil, errors.New("unrecognized format of git directory listing")
		}

		nodeType, nodeMode, err := parseTreeNodeMode(columns[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse git mode: %w", err)
		}

		nodeSha := columns[3]
		nodePath := columns[4]
		nodeName := path.Base(nodePath)

		list = append(list, types.TreeNode{
			NodeType: nodeType,
			Mode:     nodeMode,
			Sha:      nodeSha,
			Name:     nodeName,
			Path:     nodePath,
		})
	}

	return list, nil
}

// lsFile returns all tree node entries in the requested directory.
func lsDirectory(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
) ([]types.TreeNode, error) {
	treePath = path.Clean(treePath)
	if treePath == "" {
		treePath = "."
	} else {
		treePath += "/"
	}

	return lsTree(ctx, repoPath, rev, treePath)
}

// lsFile returns one tree node entry.
func lsFile(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
) (types.TreeNode, error) {
	treePath = cleanTreePath(treePath)

	list, err := lsTree(ctx, repoPath, rev, treePath)
	if err != nil {
		return types.TreeNode{}, fmt.Errorf("failed to ls file: %w", err)
	}
	if len(list) != 1 {
		return types.TreeNode{}, fmt.Errorf("ls file list contains more than one element, len=%d", len(list))
	}

	return list[0], nil
}

// GetTreeNode returns the tree node at the given path as found for the provided reference.
func (a Adapter) GetTreeNode(ctx context.Context, repoPath, rev, treePath string) (*types.TreeNode, error) {
	// root path (empty path) is a special case
	if treePath == "" {
		if repoPath == "" {
			return nil, ErrRepositoryPathEmpty
		}

		args := []string{"show", "--no-patch", "--format=" + fmtTreeHash, rev}
		treeSHA, stderr, err := gitea.NewCommand(ctx, args...).RunStdString(&gitea.RunOpts{Dir: repoPath})
		if strings.Contains(stderr, "ambiguous argument") {
			return nil, errors.NotFound("could not resolve git revision: %s", rev)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get root tree node: %w", err)
		}

		return &types.TreeNode{
			NodeType: types.TreeNodeTypeTree,
			Mode:     types.TreeNodeModeTree,
			Sha:      treeSHA,
			Name:     "",
			Path:     "",
		}, err
	}

	treeNode, err := lsFile(ctx, repoPath, rev, treePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree node: %w", err)
	}

	return &treeNode, nil
}

// ListTreeNodes lists the child nodes of a tree reachable from ref via the specified path.
func (a Adapter) ListTreeNodes(ctx context.Context, repoPath, rev, treePath string) ([]types.TreeNode, error) {
	list, err := lsDirectory(ctx, repoPath, rev, treePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list tree nodes: %w", err)
	}

	return list, nil
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
