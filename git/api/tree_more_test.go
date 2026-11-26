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
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/types"

	"github.com/stretchr/testify/require"
)

// treeRepo creates: README.md, run.sh (exec), link (symlink), single/only/leaf.txt, multi/a.txt, multi/b.txt.
func treeRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)

	repo := initRepoWithCommit(t)
	writeRepoFile(t, repo, "README.md", "readme\n")
	writeRepoFile(t, repo, "run.sh", "#!/bin/sh\n")
	require.NoError(t, chmodExec(repo, "run.sh"))
	require.NoError(t, os.Symlink("README.md", filepath.Join(repo, "link")))
	writeRepoFile(t, repo, "single/only/leaf.txt", "leaf\n")
	writeRepoFile(t, repo, "multi/a.txt", "a\n")
	writeRepoFile(t, repo, "multi/b.txt", "bb\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "tree")
	return repo
}

func nodeNames(nodes []TreeNode) []string {
	names := make([]string, len(nodes))
	for i := range nodes {
		names[i] = nodes[i].Name
	}
	sort.Strings(names)
	return names
}

func TestTreeNodeHelpers(t *testing.T) {
	require.True(t, (&TreeNode{Mode: TreeNodeModeExec}).IsExecutable())
	require.False(t, (&TreeNode{Mode: TreeNodeModeFile}).IsExecutable())
	require.True(t, (&TreeNode{Mode: TreeNodeModeTree}).IsDir())
	require.True(t, (&TreeNode{Mode: TreeNodeModeSymlink}).IsLink())
	require.True(t, (&TreeNode{Mode: TreeNodeModeCommit}).IsSubmodule())

	modes := map[TreeNodeMode]string{
		TreeNodeModeFile:    "100644",
		TreeNodeModeSymlink: "120000",
		TreeNodeModeExec:    "100755",
		TreeNodeModeTree:    "40000",
		TreeNodeModeCommit:  "160000",
	}
	for mode, want := range modes {
		require.Equal(t, want, mode.String())
	}

	for raw, want := range map[string]struct {
		typ  TreeNodeType
		mode TreeNodeMode
	}{
		"100644": {TreeNodeTypeBlob, TreeNodeModeFile},
		"120000": {TreeNodeTypeBlob, TreeNodeModeSymlink},
		"100755": {TreeNodeTypeBlob, TreeNodeModeExec},
		"160000": {TreeNodeTypeCommit, TreeNodeModeCommit},
		"040000": {TreeNodeTypeTree, TreeNodeModeTree},
	} {
		typ, mode, err := parseTreeNodeMode(raw)
		require.NoError(t, err)
		require.Equal(t, want.typ, typ)
		require.Equal(t, want.mode, mode)
	}
	_, _, err := parseTreeNodeMode("777777")
	require.Error(t, err)

	require.Equal(t, "a/b", cleanTreePath("/a/b/"))
	require.Equal(t, "b", cleanTreePath("../../b"))
	require.Empty(t, cleanTreePath(""))
}

func TestListTreeNodes(t *testing.T) {
	g := &Git{}
	repo := treeRepo(t)

	t.Run("root", func(t *testing.T) {
		nodes, err := g.ListTreeNodes(t.Context(), repo, "main", "", false)
		require.NoError(t, err)
		require.Equal(t, []string{"README.md", "a.txt", "link", "multi", "run.sh", "single"}, nodeNames(nodes))

		byName := map[string]TreeNode{}
		for _, n := range nodes {
			byName[n.Name] = n
		}
		require.Equal(t, TreeNodeModeExec, byName["run.sh"].Mode)
		require.Equal(t, TreeNodeModeSymlink, byName["link"].Mode)
		require.Equal(t, TreeNodeModeTree, byName["multi"].Mode)
		require.Equal(t, "multi", byName["multi"].Path)
	})

	t.Run("flatten single child directories", func(t *testing.T) {
		nodes, err := g.ListTreeNodes(t.Context(), repo, "main", "", true)
		require.NoError(t, err)
		require.Contains(t, nodeNames(nodes), "single/only")
		require.Contains(t, nodeNames(nodes), "multi")
	})

	t.Run("with sizes", func(t *testing.T) {
		nodes, err := ListTreeNodes(t.Context(), repo, "main", "multi", true, false)
		require.NoError(t, err)
		require.Len(t, nodes, 2)
		for _, n := range nodes {
			switch n.Name {
			case "a.txt":
				require.Equal(t, int64(2), n.Size)
			case "b.txt":
				require.Equal(t, int64(3), n.Size)
			}
		}
	})

	t.Run("recursive", func(t *testing.T) {
		nodes, err := ListTreeNodesRecursive(t.Context(), repo, "main", "", false, false)
		require.NoError(t, err)
		var paths []string
		for _, n := range nodes {
			paths = append(paths, n.Path)
		}
		require.Contains(t, paths, "multi/a.txt")
		require.Contains(t, paths, "single/only/leaf.txt")
	})

	t.Run("errors", func(t *testing.T) {
		_, err := g.ListTreeNodes(t.Context(), repo, "main", "does/not/exist", false)
		require.Error(t, err)
		_, err = g.ListTreeNodes(t.Context(), repo, "no-such-rev", "", false)
		require.Error(t, err)
		_, err = ListTreeNodesRecursive(t.Context(), repo, "no-such-rev", "", false, false)
		require.Error(t, err)
	})
}

func TestGetTreeNode_Paths(t *testing.T) {
	g := &Git{}
	repo := treeRepo(t)

	_, err := g.GetTreeNode(t.Context(), "", nil, "main", "a.txt")
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	node, err := g.GetTreeNode(t.Context(), repo, nil, "main", "multi/a.txt")
	require.NoError(t, err)
	require.Equal(t, TreeNodeTypeBlob, node.NodeType)
	require.Equal(t, "a.txt", node.Name)
	require.Equal(t, "multi/a.txt", node.Path)

	node, err = GetTreeNode(t.Context(), repo, "main", "multi/b.txt", true)
	require.NoError(t, err)
	require.Equal(t, int64(3), node.Size)

	node, err = g.GetTreeNode(t.Context(), repo, nil, "main", "multi")
	require.NoError(t, err)
	require.Equal(t, TreeNodeTypeTree, node.NodeType)

	node, err = g.GetTreeNode(t.Context(), repo, nil, "main", "")
	require.NoError(t, err)
	require.Equal(t, TreeNodeTypeTree, node.NodeType)
	require.Equal(t, runGit(t, repo, "rev-parse", "main^{tree}"), node.SHA.String())

	_, err = g.GetTreeNode(t.Context(), repo, nil, "main", "missing.txt")
	require.Error(t, err)
	_, err = g.GetTreeNode(t.Context(), repo, nil, "no-such-rev", "")
	require.Error(t, err)

	tree := runGit(t, repo, "rev-parse", "main^{tree}")
	_, err = g.GetTreeNode(t.Context(), repo, nil, tree, "")
	require.Error(t, err)
}

func TestListPaths(t *testing.T) {
	g := &Git{}
	repo := treeRepo(t)

	_, _, err := g.ListPaths(t.Context(), "", "main", false)
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	files, dirs, err := g.ListPaths(t.Context(), repo, "main", false)
	require.NoError(t, err)
	require.Empty(t, dirs)
	require.ElementsMatch(t,
		[]string{"README.md", "a.txt", "link", "multi/a.txt", "multi/b.txt", "run.sh", "single/only/leaf.txt"}, files)

	files, dirs, err = g.ListPaths(t.Context(), repo, "main", true)
	require.NoError(t, err)
	require.Contains(t, files, "multi/a.txt")
	require.ElementsMatch(t, []string{"multi", "single", "single/only"}, dirs)

	_, _, err = g.ListPaths(t.Context(), repo, "no-such-rev", false)
	require.Error(t, err)

	tree := runGit(t, repo, "rev-parse", "main^{tree}")
	_, _, err = g.ListPaths(t.Context(), repo, tree, false)
	require.Error(t, err)
}

func TestReadTree(t *testing.T) {
	g := &Git{}
	repo := treeRepo(t)

	require.ErrorIs(t, g.ReadTree(t.Context(), "", "main", io.Discard), ErrRepositoryPathEmpty)
	require.NoError(t, g.ReadTree(t.Context(), repo, "main", io.Discard))
	require.Error(t, g.ReadTree(t.Context(), repo, "main", io.Discard, "-m"))
	require.Error(t, g.ReadTree(t.Context(), repo, "no-such-rev", io.Discard))
}

func TestCleanUploadFileName(t *testing.T) {
	tests := map[string]string{
		"file.txt":         "file.txt",
		"/dir/../file.txt": "file.txt",
		"a/b/c.txt":        "a/b/c.txt",
		".git/config":      "",
		"a/.GIT/hooks":     "",
		"a/.gitignore":     "a/.gitignore",
		"":                 "",
	}
	for in, want := range tests {
		require.Equal(t, want, CleanUploadFileName(in), in)
	}
}

func TestSignatureAndIdentity(t *testing.T) {
	id := Identity{Name: "Jane", Email: "jane@test.test"}
	require.Equal(t, "Jane <jane@test.test>", id.String())
	require.Equal(t, "Jane <jane@test.test>", Signature{Identity: id, When: time.Now()}.String())

	require.NoError(t, id.Validate())
	require.Error(t, Identity{Email: "x"}.Validate())
	require.Error(t, Identity{Name: "x"}.Validate())
}

func TestNew(t *testing.T) {
	cache := NoLastCommitCache()
	var factory hook.ClientFactory
	g, err := New(types.Config{Trace: true}, cache, factory)
	require.NoError(t, err)
	require.True(t, g.traceGit)

	g, err = New(types.Config{}, cache, factory)
	require.NoError(t, err)
	require.False(t, g.traceGit)
}

func TestArchive_Formats(t *testing.T) {
	g := &Git{}
	repo := treeRepo(t)

	t.Run("param validation", func(t *testing.T) {
		require.Error(t, (&ArchiveParams{Format: ArchiveFormatTar}).Validate())
		require.Error(t, (&ArchiveParams{Treeish: "main", Format: "rar"}).Validate())
		require.NoError(t, (&ArchiveParams{Treeish: "main", Format: ArchiveFormatZip}).Validate())

		require.NoError(t, ArchiveAttributeExportIgnore.Validate())
		require.NoError(t, ArchiveAttributeExportSubst.Validate())
		require.Error(t, ArchiveAttribute("nope").Validate())

		for _, f := range ArchiveFormats {
			got, err := ParseArchiveFormat(string(f))
			require.NoError(t, err)
			require.Equal(t, f, got)
		}
		_, err := ParseArchiveFormat("rar")
		require.Error(t, err)

		var buf bytes.Buffer
		require.Error(t, g.Archive(t.Context(), repo, ArchiveParams{}, &buf))
		require.Error(t, g.Archive(t.Context(), repo, ArchiveParams{
			Treeish: "main", Format: ArchiveFormatTar, Attributes: "bad",
		}, &buf))
	})

	t.Run("tar with prefix and paths", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, g.Archive(t.Context(), repo, ArchiveParams{
			Treeish: "main", Format: ArchiveFormatTar, Prefix: "pre", Paths: []string{"multi"},
		}, &buf))

		var names []string
		tr := tar.NewReader(&buf)
		for {
			h, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoError(t, err)
			names = append(names, h.Name)
		}
		require.Contains(t, names, "pre/multi/a.txt")
		require.NotContains(t, names, "pre/README.md")
	})

	t.Run("other formats", func(t *testing.T) {
		level := 3
		ts := time.Unix(1700000000, 0).UTC()
		for _, params := range []ArchiveParams{
			{Treeish: "main", Format: ArchiveFormatZip, Compression: &level},
			{Treeish: "main", Format: ArchiveFormatTarGz, Compression: &level},
			{Treeish: "main", Format: ArchiveFormatTgz},
			{Treeish: "main", Format: ArchiveFormatTar, Compression: &level, Time: &ts},
		} {
			var buf bytes.Buffer
			require.NoError(t, g.Archive(t.Context(), repo, params, &buf), "%+v", params)
			require.NotZero(t, buf.Len())
		}
	})

	t.Run("output file", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "out.tar")
		require.NoError(t, g.Archive(t.Context(), repo, ArchiveParams{
			Treeish: "main", Format: ArchiveFormatTar, File: out,
		}, io.Discard))
		require.FileExists(t, out)
	})

	t.Run("invalid zip compression", func(t *testing.T) {
		level := 12
		require.Error(t, g.Archive(t.Context(), repo, ArchiveParams{
			Treeish: "main", Format: ArchiveFormatZip, Compression: &level,
		}, io.Discard))
	})
}
