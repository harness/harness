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
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"

	"github.com/rs/zerolog/log"
)

type TreeNodeWithCommit struct {
	TreeNode
	Commit *Commit
}

type TreeNode struct {
	NodeType TreeNodeType
	Mode     TreeNodeMode
	Sha      string
	Name     string
	Path     string
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

// Tree represents a flat directory listing.
type Tree struct {
	ID         SHA
	ResolvedID SHA

	// parent tree
	ptree *Tree

	entries       Entries
	entriesParsed bool

	entriesRecursive       Entries
	entriesRecursiveParsed bool
}

// Entries a list of entry
type Entries []*TreeEntry

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

// regexpLsTreeColumns is a regular expression that is used to parse a single line
// of a "git ls-tree" output (which uses the NULL character as the line break).
// The single line mode must be used because output might contain the EOL and other control characters.
var regexpLsTreeColumns = regexp.MustCompile(`(?s)^(\d{6})\s+(\w+)\s+(\w+)\t(.+)`)

func lsTree(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
) ([]TreeNode, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	cmd := command.New("ls-tree",
		command.WithFlag("-z"),
		command.WithArg(rev),
		command.WithArg(treePath),
	)
	output := &bytes.Buffer{}
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(output),
	)
	if err != nil {
		if strings.Contains(err.Error(), "fatal: Not a valid object name") {
			return nil, errors.NotFound("revision %q not found", rev)
		}
		return nil, fmt.Errorf("failed to run git ls-tree: %w", err)
	}

	if output.Len() == 0 {
		return nil, &PathNotFoundError{Path: treePath}
	}

	n := bytes.Count(output.Bytes(), []byte{'\x00'})

	list := make([]TreeNode, 0, n)
	scan := bufio.NewScanner(output)
	scan.Split(scanZeroSeparated)
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

		nodeSha := columns[3]
		nodePath := columns[4]
		nodeName := path.Base(nodePath)

		list = append(list, TreeNode{
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
) ([]TreeNode, error) {
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
) (TreeNode, error) {
	treePath = cleanTreePath(treePath)

	list, err := lsTree(ctx, repoPath, rev, treePath)
	if err != nil {
		return TreeNode{}, fmt.Errorf("failed to ls file: %w", err)
	}
	if len(list) != 1 {
		return TreeNode{}, fmt.Errorf("ls file list contains more than one element, len=%d", len(list))
	}

	return list[0], nil
}

// GetTreeNode returns the tree node at the given path as found for the provided reference.
func (g *Git) GetTreeNode(ctx context.Context, repoPath, rev, treePath string) (*TreeNode, error) {
	// root path (empty path) is a special case
	if treePath == "" {
		if repoPath == "" {
			return nil, ErrRepositoryPathEmpty
		}
		cmd := command.New("show",
			command.WithFlag("--no-patch"),
			command.WithFlag("--format="+fmtTreeHash),
			command.WithArg(rev),
		)
		output := &bytes.Buffer{}
		err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output))
		if err != nil {
			if strings.Contains(err.Error(), "ambiguous argument") {
				return nil, errors.NotFound("could not resolve git revision: %s", rev)
			}
			return nil, fmt.Errorf("failed to get root tree node: %w", err)
		}

		return &TreeNode{
			NodeType: TreeNodeTypeTree,
			Mode:     TreeNodeModeTree,
			Sha:      strings.TrimSpace(output.String()),
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
func (g *Git) ListTreeNodes(ctx context.Context, repoPath, rev, treePath string) ([]TreeNode, error) {
	list, err := lsDirectory(ctx, repoPath, rev, treePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list tree nodes: %w", err)
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
