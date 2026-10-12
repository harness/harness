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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// commitRenameToQuarantine renames a.txt to b.txt in a pure-rename commit whose new
// objects are written to a separate object directory, mirroring the quarantine that
// git receive-pack sets up while the pre-receive hook runs. It returns the new commit
// SHA, the blob SHA of the renamed file and the quarantine dir.
func commitRenameToQuarantine(t *testing.T, repoPath string) (string, string, string) {
	t.Helper()

	quarantineDir := t.TempDir()

	run := func(args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = repoPath
		cmd.Env = append(cmd.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_OBJECT_DIRECTORY="+quarantineDir,
			"GIT_ALTERNATE_OBJECT_DIRECTORIES="+filepath.Join(repoPath, ".git", "objects"),
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	run("mv", "a.txt", "b.txt")
	run("commit", "-m", "rename a.txt to b.txt")

	return run("rev-parse", "HEAD"), run("rev-parse", "HEAD:b.txt"), quarantineDir
}

// TestGetBlobSHAsForPaths_Quarantine asserts that blob SHAs of renamed files resolve
// against objects that live only in the push quarantine directory.
//
// Regression guard for CODE-6423: the rename SHA resolution added there ran ls-tree
// without the alternate object dirs, so during pre-receive the pushed commit was
// invisible to it. That failed the diff stream with "fatal: not a tree object", which
// aborted the vulnerability scan and rejected the push with a bare HTTP 500.
func TestGetBlobSHAsForPaths_Quarantine(t *testing.T) {
	requireGit(t)

	repoPath := initRepoWithCommit(t)
	commitSHA, blobSHA, quarantineDir := commitRenameToQuarantine(t, repoPath)

	g := &Git{}

	// Guards that the setup genuinely quarantines the new objects - without it the
	// alternates case below could pass even if they had leaked into the main store.
	if _, err := g.GetBlobSHAsForPaths(
		t.Context(), repoPath, nil, commitSHA, []string{"b.txt"},
	); err == nil {
		t.Fatal("expected ls-tree to fail without the quarantine dir")
	}

	pathToSHA, err := g.GetBlobSHAsForPaths(
		t.Context(), repoPath, []string{quarantineDir}, commitSHA, []string{"b.txt"},
	)
	if err != nil {
		t.Fatalf("GetBlobSHAsForPaths with alternates: %v", err)
	}

	if got := pathToSHA["b.txt"]; got != blobSHA {
		t.Errorf("blob sha for b.txt = %q, want %q", got, blobSHA)
	}
}
