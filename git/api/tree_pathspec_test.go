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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newTestRepoWithColonDir creates a repository whose root holds a directory named ":"
// next to other entries:
//
//	:/f.txt
//	other/g.txt
//	rootfile.txt
//
// The extra root entries matter: a path starting with ":" is magic pathspec syntax for
// "the repository root", so without --literal-pathspecs git answers with the whole root
// listing instead of the single requested entry.
func newTestRepoWithColonDir(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = repoPath
		cmd.Env = append(cmd.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@harness.io",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@harness.io",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	writeFile := func(relPath, content string) {
		t.Helper()
		fullPath := filepath.Join(repoPath, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
			t.Fatalf("failed to create dir for %q: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write %q: %v", relPath, err)
		}
	}

	run("init", "-b", "main")
	writeFile(":/f.txt", "content of the file in the colon directory\n")
	writeFile("other/g.txt", "content of g\n")
	writeFile("rootfile.txt", "content of the root file\n")
	run("add", "-A")
	run("commit", "-m", "init")

	return repoPath
}

// TestGetTreeNode_ColonDirectory asserts that paths inside a directory named ":" resolve
// to the requested entry.
//
// Regression guard for CODE-6237: paths are passed to git as pathspecs, and git reads a
// leading ":" as pathspec magic ("from the repository root"). For the directory ":" that
// made `git ls-tree <rev> :` return every root entry (so lsFile failed with "ls file list
// contains more than one element") and `git ls-tree <rev> :/f.txt` return nothing (so the
// existing file looked missing).
func TestGetTreeNode_ColonDirectory(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepoWithColonDir(t)

	t.Run("the colon directory itself", func(t *testing.T) {
		node, err := GetTreeNode(t.Context(), repoPath, "main", ":", false)
		if err != nil {
			t.Fatalf("expected to resolve the %q directory, got: %v", ":", err)
		}
		if node.Path != ":" {
			t.Errorf("expected path %q, got %q", ":", node.Path)
		}
		if !node.IsDir() {
			t.Errorf("expected %q to be a directory, got node type %v", ":", node.NodeType)
		}
	})

	t.Run("file inside the colon directory", func(t *testing.T) {
		node, err := GetTreeNode(t.Context(), repoPath, "main", ":/f.txt", false)
		if err != nil {
			t.Fatalf("expected to resolve %q, got: %v", ":/f.txt", err)
		}
		if node.Path != ":/f.txt" {
			t.Errorf("expected path %q, got %q", ":/f.txt", node.Path)
		}
		if node.NodeType != TreeNodeTypeBlob {
			t.Errorf("expected %q to be a blob, got node type %v", ":/f.txt", node.NodeType)
		}
	})

	t.Run("missing file inside the colon directory", func(t *testing.T) {
		_, err := GetTreeNode(t.Context(), repoPath, "main", ":/missing.txt", false)
		if err == nil {
			t.Fatalf("expected a not found error for %q, got nil", ":/missing.txt")
		}
		if !strings.Contains(err.Error(), "wasn't found") {
			t.Errorf("expected a not found error for %q, got: %v", ":/missing.txt", err)
		}
	})
}

// TestListTreeNodes_ColonDirectory asserts that listing a directory named ":" returns its
// children rather than the repository root listing.
//
// Regression guard for CODE-6237.
func TestListTreeNodes_ColonDirectory(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepoWithColonDir(t)

	nodes, err := ListTreeNodes(t.Context(), repoPath, "main", ":", false, false)
	if err != nil {
		t.Fatalf("expected to list the %q directory, got: %v", ":", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected exactly one child of %q, got %d: %+v", ":", len(nodes), nodes)
	}
	if nodes[0].Path != ":/f.txt" {
		t.Errorf("expected child %q, got %q", ":/f.txt", nodes[0].Path)
	}
}
