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

package sharedrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/errors"

	"github.com/stretchr/testify/require"
)

// TestApplyToIndex_FailedApplyReturnsConflict verifies that a patch which does not apply
// is surfaced as a conflict (HTTP 409) error carrying the git stderr, rather than a
// generic internal error that hides the reason.
func TestApplyToIndex_FailedApplyReturnsConflict(t *testing.T) {
	ctx := context.Background()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init", "-q")
	runGit(t, repoDir, "config", "user.email", "test@harness.io")
	runGit(t, repoDir, "config", "user.name", "test")

	// Commit a file so there is tracked content the patch can fail against.
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello\n"), 0o600))
	runGit(t, repoDir, "add", "file.txt")
	runGit(t, repoDir, "commit", "-q", "-m", "init")

	// Patch context ("world") does not match the committed content ("hello"), so
	// "git apply --cached" fails with "patch does not apply".
	patch := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\n" +
		"+++ b/file.txt\n" +
		"@@ -1 +1 @@\n" +
		"-world\n" +
		"+changed\n"
	patchFile := filepath.Join(t.TempDir(), "bad.patch")
	require.NoError(t, os.WriteFile(patchFile, []byte(patch), 0o600))

	r := &SharedRepo{repoPath: repoDir}

	err := r.ApplyToIndex(ctx, patchFile)
	require.Error(t, err)

	var appErr *errors.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, errors.StatusConflict, appErr.Status)
	require.Contains(t, appErr.Message, "does not apply")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v failed: %s", args, out)
}
