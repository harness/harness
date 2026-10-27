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
	"testing"

	"github.com/harness/gitness/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestArchive_UnknownTreeish asserts that archiving a missing revision returns NotFound.
func TestArchive_UnknownTreeish(t *testing.T) {
	requireGit(t)

	emptyRepoPath := t.TempDir()
	out, err := exec.CommandContext(t.Context(), "git", "init", "--initial-branch", "main", emptyRepoPath).
		CombinedOutput()
	require.NoErrorf(t, err, "git init failed: %s", out)

	repoPath := initRepoWithCommit(t)

	tests := []struct {
		name     string
		repoPath string
		treeish  string
	}{
		{name: "HEAD in empty repo", repoPath: emptyRepoPath, treeish: "HEAD"},
		{name: "branch in empty repo", repoPath: emptyRepoPath, treeish: "refs/heads/main"},
		{name: "unknown branch", repoPath: repoPath, treeish: "refs/heads/does-not-exist"},
		{name: "unknown short name", repoPath: repoPath, treeish: "does-not-exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := ArchiveParams{Format: ArchiveFormatTar, Treeish: tt.treeish}
			err := (&Git{}).Archive(t.Context(), tt.repoPath, params, &bytes.Buffer{})
			require.Error(t, err)
			assert.True(t, errors.IsNotFound(err), "expected not found, got: %v", err)
		})
	}
}

func TestArchive_ExistingTreeish(t *testing.T) {
	requireGit(t)

	repoPath := initRepoWithCommit(t)
	_, err := os.Stat(repoPath)
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	params := ArchiveParams{Format: ArchiveFormatTar, Treeish: "HEAD"}
	require.NoError(t, (&Git{}).Archive(t.Context(), repoPath, params, buf))
	assert.NotZero(t, buf.Len())
}
