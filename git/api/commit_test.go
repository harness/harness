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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/errors"
)

// initRepoWithCommit creates a throw-away repo on branch main with a single commit.
func initRepoWithCommit(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = repoPath
		cmd.Env = append(cmd.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	run("init", "--initial-branch", "main", ".")
	run("config", "user.email", "test@test.test")
	run("config", "user.name", "test")

	if err := os.WriteFile(filepath.Join(repoPath, "a.txt"), []byte("a\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	run("add", "a.txt")
	run("commit", "-m", "first")

	return repoPath
}

// TestListCommitSHAs_NonExistentAfterRef asserts that a non-existent "after" ref
// resolves to an empty result rather than an internal error.
//
// Regression guard for CODE-5125: git reports a bad *positional* revision as
// "ambiguous argument" and a bad full SHA as "bad object", but a bad *named*
// revision behind the "^" exclusion prefix as "bad revision". Only the first two
// were recognized, so `after=<unknown-branch>` produced an errors.StatusInternal,
// which gitrpc mapped to codes.Internal, which githa classifies as an
// intermittent error - burning every replica and surfacing an HTTP 503 for what
// is really a bad client request.
func TestListCommitSHAs_NonExistentAfterRef(t *testing.T) {
	requireGit(t)

	repoPath := initRepoWithCommit(t)
	g := &Git{}

	// "undefined" is the literal value observed in production (a UI sending
	// String(undefined)); the bare and fully-qualified forms take the same path.
	for _, afterRef := range []string{"undefined", "refs/heads/undefined"} {
		t.Run(afterRef, func(t *testing.T) {
			shas, err := g.listCommitSHAs(
				context.Background(),
				repoPath,
				nil,
				"main",
				1,
				10,
				CommitFilter{AfterRef: afterRef},
			)
			if err != nil {
				t.Fatalf("expected no error for non-existent after ref %q, got status=%q err=%v",
					afterRef, errors.AsStatus(err), err)
			}
			if len(shas) != 0 {
				t.Fatalf("expected empty commit list, got %d", len(shas))
			}
		})
	}
}

// TestGetCommitDivergences_NonExistentRef asserts that an unknown ref yields the
// documented "unknown" divergence sentinel (-1, -1) instead of an internal error.
//
// Regression guard for CODE-5125: GetCommitDivergences already has a branch for
// errors.IsNotFound, but getCommitDivergence never produced NotFound for an
// unknown ref - git's "ambiguous argument" fell through processGitErrorf's
// default case to StatusInternal, making that branch dead code and turning a bad
// client ref into a 503. This is the second page (page > 1) of the same
// list-commits request that trips TestListCommitSHAs_NonExistentAfterRef.
func TestGetCommitDivergences_NonExistentRef(t *testing.T) {
	requireGit(t)

	repoPath := initRepoWithCommit(t)
	g := &Git{}

	res, err := g.GetCommitDivergences(
		context.Background(),
		repoPath,
		[]CommitDivergenceRequest{{From: "main", To: "undefined"}},
		0,
	)
	if err != nil {
		t.Fatalf("expected no error for non-existent ref, got status=%q err=%v", errors.AsStatus(err), err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 divergence result, got %d", len(res))
	}
	if res[0].Ahead != -1 || res[0].Behind != -1 {
		t.Fatalf("expected unknown divergence (-1, -1), got (%d, %d)", res[0].Ahead, res[0].Behind)
	}
}
