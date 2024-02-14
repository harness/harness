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
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/types"

	"github.com/rs/zerolog/log"
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

// regexpLsTreeColumns is a regular expression that is used to parse a single line
// of a "git ls-tree" output (which uses the NULL character as the line break).
// The single line mode must be used because output might contain the EOL and other control characters.
var regexpLsTreeColumns = regexp.MustCompile(`(?s)^(\d{6})\s+(\w+)\s+(\w+)\t(.+)`)

func lsTree(
	ctx context.Context,
	repoPath string,
	rev string,
	treePath string,
) ([]types.TreeNode, error) {
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
		return nil, &types.PathNotFoundError{Path: treePath}
	}

	n := bytes.Count(output.Bytes(), []byte{'\x00'})

	list := make([]types.TreeNode, 0, n)
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

		return &types.TreeNode{
			NodeType: types.TreeNodeTypeTree,
			Mode:     types.TreeNodeModeTree,
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
	cmd := command.New("read-tree",
		command.WithArg(ref),
		command.WithArg(args...),
	)
	if err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(w)); err != nil {
		return fmt.Errorf("unable to read %s in to the index: %w", ref, err)
	}
	return nil
}
