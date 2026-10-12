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

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harness/gitness/git/api"
)

// initRepoWithQuarantinedRename builds a repo under reposRoot holding a base commit and a
// pure-rename commit whose new objects are written to a separate object directory, the way
// git receive-pack quarantines a push while the pre-receive hook runs. It returns the base
// commit SHA, the rename commit SHA and the quarantine dir.
func initRepoWithQuarantinedRename(t *testing.T, reposRoot, repoUID string) (string, string, string) {
	t.Helper()

	repoPath := getFullPathForRepo(reposRoot, repoUID)
	if err := os.MkdirAll(repoPath, 0o700); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	quarantineDir := t.TempDir()

	run := func(quarantined bool, args ...string) string {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = repoPath
		cmd.Env = append(cmd.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		)
		if quarantined {
			cmd.Env = append(cmd.Env,
				"GIT_OBJECT_DIRECTORY="+quarantineDir,
				"GIT_ALTERNATE_OBJECT_DIRECTORIES="+filepath.Join(repoPath, ".git", "objects"),
			)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	run(false, "init", "--initial-branch", "main", ".")
	run(false, "config", "user.email", "test@test.test")
	run(false, "config", "user.name", "test")

	if err := os.WriteFile(filepath.Join(repoPath, "a.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run(false, "add", "a.txt")
	run(false, "commit", "-m", "first")
	baseSHA := run(false, "rev-parse", "HEAD")

	// The rename is byte-identical, so git records it as a pure rename with no index line
	// and writes only tree and commit objects - into the quarantine dir.
	run(true, "mv", "a.txt", "b.txt")
	run(true, "commit", "-m", "rename a.txt to b.txt")
	headSHA := run(true, "rev-parse", "HEAD")

	return baseSHA, headSHA, quarantineDir
}

// TestDiff_PureRenameInQuarantine asserts that diffing a pushed pure rename succeeds while
// the new objects are still quarantined, which is the state every pre-receive hook sees.
//
// Regression guard for CODE-6423: the rename SHA resolution added there ran ls-tree without
// the alternate object dirs, so the pushed commit was invisible to it. The diff stream died
// with "fatal: not a tree object", aborting the pre-receive vulnerability scan and rejecting
// the push with a bare HTTP 500.
func TestDiff_PureRenameInQuarantine(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	reposRoot := t.TempDir()
	const repoUID = "testrepo-pure-rename"

	baseSHA, headSHA, quarantineDir := initRepoWithQuarantinedRename(t, reposRoot, repoUID)

	s := &Service{reposRoot: reposRoot, git: &api.Git{}}

	chFiles, chErr := s.Diff(t.Context(), &DiffParams{
		ReadParams: ReadParams{
			RepoUID:             repoUID,
			AlternateObjectDirs: []string{quarantineDir},
		},
		BaseRef: baseSHA,
		HeadRef: headSHA,
	})

	var files []*FileDiff
	for f := range chFiles {
		files = append(files, f)
	}
	if err := <-chErr; err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d file diffs, want 1", len(files))
	}
	if files[0].Path != "b.txt" {
		t.Errorf("path = %q, want %q", files[0].Path, "b.txt")
	}
	if files[0].OldPath != "a.txt" {
		t.Errorf("old path = %q, want %q", files[0].OldPath, "a.txt")
	}
}
