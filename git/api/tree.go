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

package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"

	"github.com/rs/zerolog/log"
)

type TreeNodeWithCommit struct {
	TreeNode
	Commit *Commit
}

type TreeNode struct {
	NodeType TreeNodeType
	Mode     TreeNodeMode
	SHA      sha.SHA
	Name     string
	Path     string
	Size     int64
}

func (n *TreeNode) IsExecutable() bool {
	return n.Mode == TreeNodeModeExec
}

func (n *TreeNode) IsDir() bool {
	return n.Mode == TreeNodeModeTree
}

func (n *TreeNode) IsLink() bool {
	return n.Mode == TreeNodeModeSymlink
}

func (n *TreeNode) IsSubmodule() bool {
	return n.Mode == TreeNodeModeCommit
}

// TreeNodeType specifies the different types of nodes in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeType (proto).
type TreeNodeType int

const (
	TreeNodeTypeTree TreeNodeType = iota
	TreeNodeTypeBlob
	TreeNodeTypeCommit
)

// TreeNodeMode specifies the different modes of a node in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeMode (proto).
type TreeNodeMode int

const (
	TreeNodeModeFile TreeNodeMode = iota
	TreeNodeModeSymlink
	TreeNodeModeExec
	TreeNodeModeTree
	TreeNodeModeCommit
)

func (m TreeNodeMode) String() string {
	var result int
	switch m {
	case TreeNodeModeFile:
		result = 0o100644
	case TreeNodeModeSymlink:
		result = 0o120000
	case TreeNodeModeExec:
		result = 0o100755
	case TreeNodeModeTree:
		result = 0o040000
	case TreeNodeModeCommit:
		result = 0o160000
	}
	return strconv.FormatInt(int64(result), 8)
}

func cleanTreePath(treePath string) string {
	return strings.Trim(path.Clean("/"+treePath), "/")
}

func parseTreeNodeMode(s string) (TreeNodeType, TreeNodeMode, error) {
	switch s {
	case "100644":
		return TreeNodeTypeBlob, TreeNodeModeFile, nil
	case "120000":
		return TreeNodeTypeBlob, TreeNodeModeSymlink, nil
	case "100755":
		return TreeNodeTypeBlob, TreeNodeModeExec, nil
	case "160000":
		return TreeNodeTypeCommit, TreeNodeModeCommit, nil
	case "040000":
		return TreeNodeTypeTree, TreeNodeModeTree, nil
	default:
		return TreeNodeTypeBlob, TreeNodeModeFile,
			fmt.Errorf("unknown git tree node mode: '%s'", s)
	}
}

// regexpLsTreeColumns is a regular expression that is used to parse a single line
// of a "git ls-tree" output (which uses the NULL character as the line break).
// The single line mode must be used because output might contain the EOL and other control characters.
var regexpLsTreeColumns = regexp.MustCompile(`(?s)^(\d{6})\s+(\w+)\s+(\w+)(?:\s+(\d+|-))?\t(.+)`)

func lsTree(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
	fetchSizes bool,
	recursive bool,
) ([]TreeNode, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	cmd := command.New("ls-tree",
		command.WithFlag("-z"),
		command.WithArg(rev),
		command.WithArg(treePath),
	)
	if fetchSizes {
		cmd.Add(command.WithFlag("-l"))
	}
	if recursive {
		cmd.Add(command.WithFlag("-r"))
	}

	output := &bytes.Buffer{}
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(output),
	)
	if err != nil {
		if strings.Contains(err.Error(), "fatal: not a tree object") {
			return nil, errors.InvalidArgumentf("revision %q does not point to a commit", rev)
		}
		if strings.Contains(err.Error(), "fatal: Not a valid object name") {
			return nil, errors.NotFoundf("revision %q not found", rev)
		}
		return nil, fmt.Errorf("failed to run git ls-tree: %w", err)
	}

	if output.Len() == 0 {
		return nil, errors.NotFoundf("path '%s' wasn't found in the repo", treePath)
	}

	n := bytes.Count(output.Bytes(), []byte{'\x00'})

	list := make([]TreeNode, 0, n)
	scan := bufio.NewScanner(output)
	scan.Split(parser.ScanZeroSeparated)
	for scan.Scan() {
		line := scan.Text()

		columns := regexpLsTreeColumns.FindStringSubmatch(line)
		if columns == nil {
			log.Ctx(ctx).Error().
				Str("ls-tree", output.String()). // logs the whole directory listing for the additional context
				Str("line", line).
				Msg("unrecognized format of git directory listing")
			return nil, fmt.Errorf("unrecognized format of git directory listing: %q", line)
		}

		nodeType, nodeMode, err := parseTreeNodeMode(columns[1])
		if err != nil {
			log.Ctx(ctx).Err(err).
				Str("line", line).
				Msg("failed to parse git mode")
			return nil, fmt.Errorf("failed to parse git node type and file mode: %w", err)
		}

		nodeSha := sha.Must(columns[3])

		var size int64
		if columns[4] != "" && columns[4] != "-" {
			size, err = strconv.ParseInt(columns[4], 10, 64)
			if err != nil {
				log.Ctx(ctx).Error().
					Str("line", line).
					Msg("failed to parse file size")
				return nil, fmt.Errorf("failed to parse file size in the git directory listing: %q", line)
			}
		}

		nodePath := columns[5]
		nodeName := path.Base(nodePath)

		list = append(list, TreeNode{
			NodeType: nodeType,
			Mode:     nodeMode,
			SHA:      nodeSha,
			Name:     nodeName,
			Path:     nodePath,
			Size:     size,
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
	fetchSizes bool,
	flattenDirectories bool,
	recursive bool,
) ([]TreeNode, error) {
	treePath = path.Clean(treePath)
	if treePath == "" {
		treePath = "."
	} else {
		treePath += "/"
	}

	nodes, err := lsTree(ctx, repoPath, rev, treePath, fetchSizes, recursive)
	if err != nil {
		return nil, err
	}

	if flattenDirectories {
		for i := range nodes {
			if nodes[i].NodeType != TreeNodeTypeTree {
				continue
			}

			if err := flattenDirectory(ctx, repoPath, rev, &nodes[i], fetchSizes); err != nil {
				return nil, fmt.Errorf("failed to flatten directory: %w", err)
			}
		}
	}

	return nodes, nil
}

// lsFile returns one tree node entry.
func lsFile(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
	fetchSize bool,
) (TreeNode, error) {
	treePath = cleanTreePath(treePath)

	list, err := lsTree(ctx, repoPath, rev, treePath, fetchSize, false)
	if err != nil {
		return TreeNode{}, fmt.Errorf("failed to ls file: %w", err)
	}
	if len(list) != 1 {
		return TreeNode{}, fmt.Errorf("ls file list contains more than one element, len=%d", len(list))
	}

	return list[0], nil
}

func flattenDirectory(
	ctx context.Context,
	repoPath string,
	rev string,
	node *TreeNode,
	fetchSizes bool,
) error {
	nodes := []TreeNode{*node}
	var pathPrefix string

	// Go in depth for as long as there are subdirectories with just one subdirectory.
	for len(nodes) == 1 && nodes[0].NodeType == TreeNodeTypeTree {
		nodesTemp, err := lsTree(ctx, repoPath, rev, nodes[0].Path+"/", fetchSizes, false)
		if err != nil {
			return fmt.Errorf("failed to peek dir entries for flattening: %w", err)
		}

		// Abort when the subdirectory contains more than one entry or contains an entry which is not a directory.
		// Git doesn't store empty directories. Every git tree must have at least one entry (except the sha.EmptyTree).
		if len(nodesTemp) != 1 || (len(nodesTemp) == 1 && nodesTemp[0].NodeType != TreeNodeTypeTree) {
			nodes[0].Name = path.Join(pathPrefix, nodes[0].Name)
			*node = nodes[0]
			break
		}

		pathPrefix = path.Join(pathPrefix, nodes[0].Name)
		nodes = nodesTemp
	}

	return nil
}

// GetTreeNode returns the tree node at the given path as found for the provided reference.
func (g *Git) GetTreeNode(ctx context.Context, repoPath, rev, treePath string) (*TreeNode, error) {
	return GetTreeNode(ctx, repoPath, rev, treePath, false)
}

// GetTreeNode returns the tree node at the given path as found for the provided reference.
func GetTreeNode(ctx context.Context, repoPath, rev, treePath string, fetchSize bool) (*TreeNode, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}

	// anything that's not the root path is a simple call
	if treePath != "" {
		treeNode, err := lsFile(ctx, repoPath, rev, treePath, fetchSize)
		if err != nil {
			return nil, fmt.Errorf("failed to get tree node: %w", err)
		}

		return &treeNode, nil
	}

	// root path (empty path) is a special case
	cmd := command.New("show",
		command.WithFlag("--no-patch"),
		command.WithFlag("--format="+fmtTreeHash), //nolint:goconst
		command.WithArg(rev+"^{commit}"),          //nolint:goconst
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
	if err != nil {
		if strings.Contains(err.Error(), "expected commit type") {
			return nil, errors.InvalidArgumentf("revision %q does not point to a commit", rev)
		}
		if strings.Contains(err.Error(), "unknown revision") {
			return nil, errors.NotFoundf("revision %q not found", rev)
		}
		return nil, fmt.Errorf("failed to get root tree node: %w", err)
	}

	return &TreeNode{
		NodeType: TreeNodeTypeTree,
		Mode:     TreeNodeModeTree,
		SHA:      sha.Must(output.String()),
		Name:     "",
		Path:     "",
	}, err
}

// ListTreeNodes lists the child nodes of a tree reachable from ref via the specified path.
func (g *Git) ListTreeNodes(
	ctx context.Context,
	repoPath, rev, treePath string,
	flattenDirectories bool,
) ([]TreeNode, error) {
	return ListTreeNodes(ctx, repoPath, rev, treePath, false, flattenDirectories)
}

// ListTreeNodes lists the child nodes of a tree reachable from ref via the specified path.
func ListTreeNodes(
	ctx context.Context,
	repoPath, rev, treePath string,
	fetchSizes, flattenDirectories bool,
) ([]TreeNode, error) {
	list, err := lsDirectory(
		ctx, repoPath, rev, treePath, fetchSizes, flattenDirectories, false,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tree nodes: %w", err)
	}

	return list, nil
}

func ListTreeNodesRecursive(
	ctx context.Context,
	repoPath, rev, treePath string,
	fetchSizes, flattenDirectories bool,
) ([]TreeNode, error) {
	list, err := lsDirectory(
		ctx, repoPath, rev, treePath, fetchSizes, flattenDirectories, true,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list tree nodes recursive: %w", err)
	}

	return list, nil
}

func (g *Git) ReadTree(
	ctx context.Context,
	repoPath string,
	ref string,
	w io.Writer,
	args ...string,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	cmd := command.New("read-tree",
		command.WithArg(ref),
		command.WithArg(args...),
	)
	if err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(w)); err != nil {
		return fmt.Errorf("unable to read %s in to the index: %w", ref, err)
	}
	return nil
}

// ListPaths lists all the paths in a repo recursively (similar-ish to `ls -lR`).
// Note: Keep method simple for now to avoid unnecessary corner cases
// by always listing whole repo content (and not relative to any directory).
func (g *Git) ListPaths(
	ctx context.Context,
	repoPath string,
	rev string,
	includeDirs bool,
) (files []string, dirs []string, err error) {
	if repoPath == "" {
		return nil, nil, ErrRepositoryPathEmpty
	}

	// use custom ls-tree for speed up (file listing is ~10% faster, with dirs roughly the same)
	cmd := command.New("ls-tree",
		command.WithConfig("core.quotePath", "false"), // force printing of path in custom format without quoting
		command.WithFlag("-z"),
		command.WithFlag("-r"),
		command.WithFlag("--full-name"),
		command.WithArg(rev+"^{commit}"), //nolint:goconst enforce commit revs for now (keep it simple)
	)
	format := fmtFieldPath
	if includeDirs {
		cmd.Add(command.WithFlag("-t"))
		format += fmtZero + fmtFieldObjectType
	}
	cmd.Add(command.WithFlag("--format=" + format)) //nolint:goconst

	output := &bytes.Buffer{}
	err = cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(output),
	)
	if err != nil {
		if strings.Contains(err.Error(), "expected commit type") {
			return nil, nil, errors.InvalidArgumentf("revision %q does not point to a commit", rev)
		}
		if strings.Contains(err.Error(), "fatal: Not a valid object name") {
			return nil, nil, errors.NotFoundf("revision %q not found", rev)
		}
		return nil, nil, fmt.Errorf("failed to run git ls-tree: %w", err)
	}

	scanner := bufio.NewScanner(output)
	scanner.Split(parser.ScanZeroSeparated)
	for scanner.Scan() {
		path := scanner.Text()

		isDir := false
		if includeDirs {
			// custom format guarantees the object type in the next scan
			if !scanner.Scan() {
				return nil, nil, fmt.Errorf("unexpected output from ls-tree when getting object type: %w", scanner.Err())
			}
			objectType := scanner.Text()
			isDir = strings.EqualFold(objectType, string(GitObjectTypeTree))
		}

		if isDir {
			dirs = append(dirs, path)
			continue
		}

		files = append(files, path)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading ls-tree output: %w", err)
	}

	return files, dirs, nil
}
