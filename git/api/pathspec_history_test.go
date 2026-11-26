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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// commitColonFileChange appends a line to ":/f.txt" and commits it, so that the history of
// the file differs from the history of the repository.
func commitColonFileChange(t *testing.T, repoPath string) {
	t.Helper()

	f, err := os.OpenFile(filepath.Join(repoPath, ":", "f.txt"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("failed to open the colon file: %v", err)
	}
	if _, err = f.WriteString("second line\n"); err != nil {
		t.Fatalf("failed to write the colon file: %v", err)
	}
	if err = f.Close(); err != nil {
		t.Fatalf("failed to close the colon file: %v", err)
	}

	for _, args := range [][]string{{"add", "-A"}, {"commit", "-m", "change the colon file"}} {
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
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			t.Fatalf("git %v: %v: %s", args, runErr, out)
		}
	}
}

// TestGetLatestCommit_ColonDirectory asserts that the history of a file inside a directory
// named ":" is looked up by its literal path.
//
// Regression guard for CODE-6237: without --literal-pathspecs git reads ":/f.txt" as "f.txt"
// in the repository root, which matches nothing.
func TestGetLatestCommit_ColonDirectory(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepoWithColonDir(t)
	commitColonFileChange(t, repoPath)

	commit, err := GetLatestCommit(t.Context(), repoPath, "main", ":/f.txt")
	if err != nil {
		t.Fatalf("expected the latest commit of %q, got: %v", ":/f.txt", err)
	}
	if commit.Title != "change the colon file" {
		t.Errorf("expected the commit that changed the file, got %q", commit.Title)
	}
}

// TestListCommitSHAs_ColonDirectory asserts that the path filter of the commit listing
// takes a path inside a directory named ":" literally.
//
// Regression guard for CODE-6237.
func TestListCommitSHAs_ColonDirectory(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepoWithColonDir(t)
	commitColonFileChange(t, repoPath)

	shas, err := (&Git{}).listCommitSHAs(t.Context(), repoPath, nil, "main", 1, 10,
		CommitFilter{Path: ":/f.txt"})
	if err != nil {
		t.Fatalf("expected to list the commits of %q, got: %v", ":/f.txt", err)
	}
	// the commit that added the file and the one that changed it
	if len(shas) != 2 {
		t.Errorf("expected 2 commits for %q, got %d", ":/f.txt", len(shas))
	}
}

// TestBlame_ColonDirectory asserts that a file inside a directory named ":" can be blamed.
//
// Regression guard for CODE-6237.
func TestBlame_ColonDirectory(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepoWithColonDir(t)
	commitColonFileChange(t, repoPath)

	reader := (&Git{}).Blame(t.Context(), repoPath, "main", ":/f.txt", 0, 0)

	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("expected to blame %q, got: %v", ":/f.txt", err)
	}
	if part == nil || len(part.Lines) == 0 {
		t.Fatalf("expected blamed lines for %q, got: %+v", ":/f.txt", part)
	}
}

// TestRawDiff_ColonDirectory asserts that a diff restricted to a file inside a directory
// named ":" contains that file.
//
// Regression guard for CODE-6237.
func TestRawDiff_ColonDirectory(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepoWithColonDir(t)
	commitColonFileChange(t, repoPath)

	out := &bytes.Buffer{}
	err := (&Git{}).RawDiff(t.Context(), out, repoPath, "main~1", "main", false, false, nil,
		FileDiffRequest{Path: ":/f.txt"})
	if err != nil {
		t.Fatalf("expected to diff %q, got: %v", ":/f.txt", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("+second line")) {
		t.Errorf("expected the change of %q in the diff, got:\n%s", ":/f.txt", out.String())
	}
}
