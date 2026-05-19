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
	"errors"
	"os"
	"os/exec"
	"testing"
)

// requireGit skips the test if the git binary is not available on PATH.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}
}

// TestListLocalReferences_NoMatch asserts that "exit 1 with empty stderr"
// from `git show-ref` (Git's documented signal for "no matching refs found")
// is treated as a valid empty result, not a fatal error.
//
// This is the regression guard for the chicken-and-egg failure where the
// very first SyncRefs call for a brand-new ref would abort because the ref
// had never been mirrored locally.
//
// The test runs against the current working directory (the test package dir,
// which is inside the gitness repo) and queries a clearly non-existent ref
// so `git show-ref` produces the "no match" outcome without needing a
// purpose-built bare repo.
func TestListLocalReferences_NoMatch(t *testing.T) {
	requireGit(t)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	g := &Git{}
	refs, err := g.ListLocalReferences(
		context.Background(),
		cwd,
		"refs/heads/__gitness_code_5450_nonexistent_ref__",
	)
	if err != nil {
		t.Fatalf("expected no error for missing ref, got: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected empty map for missing ref, got: %v", refs)
	}
}

// TestListLocalReferences_EmptyPath asserts that an empty repo path still
// returns the dedicated sentinel error - the new error swallow must not
// shadow basic input validation.
func TestListLocalReferences_EmptyPath(t *testing.T) {
	g := &Git{}
	_, err := g.ListLocalReferences(context.Background(), "", "refs/heads/main")
	if !errors.Is(err, ErrRepositoryPathEmpty) {
		t.Fatalf("expected ErrRepositoryPathEmpty, got: %v", err)
	}
}

// TestListLocalReferences_BadRepo asserts that a real git failure
// (path exists but is not a git repository -> exit 128 with stderr)
// continues to surface as an error after the patch.
func TestListLocalReferences_BadRepo(t *testing.T) {
	requireGit(t)
	notARepo := t.TempDir()

	g := &Git{}
	_, err := g.ListLocalReferences(context.Background(), notARepo, "refs/heads/main")
	if err == nil {
		t.Fatalf("expected error for non-git directory, got nil")
	}
}
