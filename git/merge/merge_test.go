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

package merge

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// multiLineMessage is a commit message with a subject and a multi-paragraph body.
// The subject must not be duplicated into the body of the rebased commit.
const multiLineMessage = `Add the answer to everything

The body of the commit message spans
multiple lines.

Signed-off-by: Test Author <author@test.io>
`

func TestRebase_PreservesCommitMessageAndAuthor(t *testing.T) {
	requireGit(t)

	ctx := context.Background()

	repoPath := newTestRepo(t)

	baseSHA := commitFile(t, repoPath, "base.txt", "base\n", "Initial commit\n")
	runGit(t, repoPath, "checkout", "-b", "feature")
	sourceSHA := commitFile(t, repoPath, "source.txt", "source\n", multiLineMessage)
	runGit(t, repoPath, "checkout", "main")
	targetSHA := commitFile(t, repoPath, "target.txt", "target\n", "Move the target\n")

	s := newSharedRepo(ctx, t, repoPath)

	committer := &api.Signature{
		Identity: api.Identity{Name: "Merge Bot", Email: "bot@test.io"},
		When:     time.Date(2021, 2, 3, 4, 5, 6, 0, time.UTC),
	}

	mergeSHA, conflicts, err := Rebase(ctx, s, Params{
		Author:       committer, // rebase must ignore the author from the params
		Committer:    committer,
		Message:      "this message must be ignored by rebase",
		MergeBaseSHA: baseSHA,
		TargetSHA:    targetSHA,
		SourceSHA:    sourceSHA,
	})
	require.NoError(t, err)
	require.Empty(t, conflicts)

	source, err := api.GetCommit(ctx, s.Directory(), sourceSHA)
	require.NoError(t, err)

	rebased, err := api.GetCommit(ctx, s.Directory(), mergeSHA)
	require.NoError(t, err)

	// The message must be carried over verbatim - in particular the subject must appear only once.
	assert.Equal(t, source.Message, rebased.Message)
	assert.Equal(t, source.Title, rebased.Title)
	assert.Equal(t, 1, strings.Count(rebased.Message, source.Title))

	// Rebase preserves the author (including the date) and sets the committer from the params.
	assert.Equal(t, source.Author, rebased.Author)
	assert.Equal(t, committer.Identity, rebased.Committer.Identity)

	// The rebased commit is stacked on top of the target.
	assert.Equal(t, []sha.SHA{targetSHA}, rebased.ParentSHAs)
}

func TestMergeAndSquash_UseParamsMessage(t *testing.T) {
	requireGit(t)

	ctx := context.Background()

	repoPath := newTestRepo(t)

	baseSHA := commitFile(t, repoPath, "base.txt", "base\n", "Initial commit\n")
	runGit(t, repoPath, "checkout", "-b", "feature")
	sourceSHA := commitFile(t, repoPath, "source.txt", "source\n", multiLineMessage)
	runGit(t, repoPath, "checkout", "main")
	targetSHA := commitFile(t, repoPath, "target.txt", "target\n", "Move the target\n")

	signature := &api.Signature{
		Identity: api.Identity{Name: "Merge Bot", Email: "bot@test.io"},
		When:     time.Date(2021, 2, 3, 4, 5, 6, 0, time.UTC),
	}

	const message = "Merge pull request #1\n\nThe body of the merge commit message.\n"

	tests := []struct {
		name        string
		fn          Func
		wantParents []sha.SHA
	}{
		{
			name:        "merge",
			fn:          Merge,
			wantParents: []sha.SHA{targetSHA, sourceSHA},
		},
		{
			name:        "squash",
			fn:          Squash,
			wantParents: []sha.SHA{targetSHA},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := newSharedRepo(ctx, t, repoPath)

			mergeSHA, conflicts, err := test.fn(ctx, s, Params{
				Author:       signature,
				Committer:    signature,
				Message:      message,
				MergeBaseSHA: baseSHA,
				TargetSHA:    targetSHA,
				SourceSHA:    sourceSHA,
			})
			require.NoError(t, err)
			require.Empty(t, conflicts)

			commit, err := api.GetCommit(ctx, s.Directory(), mergeSHA)
			require.NoError(t, err)

			assert.Equal(t, message, commit.Message)
			assert.Equal(t, "Merge pull request #1", commit.Title)
			assert.Equal(t, test.wantParents, commit.ParentSHAs)
		})
	}
}

// TestRebase_ObjectsSurvivePacking rebases enough commits for the shared repository to combine
// the created objects into a pack file (as sharedrepo.Run does) and asserts that the rebased
// branch is fully available in the original repository afterwards.
func TestRebase_ObjectsSurvivePacking(t *testing.T) {
	requireGit(t)

	ctx := context.Background()

	repoPath := newTestRepo(t)

	// The file is nested deeply, so that every rebased commit creates a blob, the commit itself
	// and a tree per path element. That way a handful of commits suffices to push the number of
	// created objects above the packing threshold of the shared repository.
	const filePath = "a/b/c/d/e/f/g/h/i/j/source.txt"

	baseContent := ""
	for i := range 20 {
		baseContent += fmt.Sprintf("line %d\n", i)
	}

	baseSHA := commitFile(t, repoPath, filePath, baseContent, "Initial commit\n")

	// the source commits append to the end of the file
	runGit(t, repoPath, "checkout", "-b", "feature")

	content := baseContent
	var sourceSHA sha.SHA
	for i := 1; i <= 9; i++ {
		content += fmt.Sprintf("appended %d\n", i)
		sourceSHA = commitFile(t, repoPath, filePath, content, fmt.Sprintf("Change %d\n", i))
	}

	// the target commit changes the beginning of the same file, so that rebasing it creates
	// new objects rather than reusing the ones of the source branch.
	runGit(t, repoPath, "checkout", "main")
	targetSHA := commitFile(t, repoPath, filePath, "prepended\n"+baseContent, "Move the target\n")

	s := newSharedRepo(ctx, t, repoPath)

	committer := &api.Signature{
		Identity: api.Identity{Name: "Merge Bot", Email: "bot@test.io"},
		When:     time.Date(2021, 2, 3, 4, 5, 6, 0, time.UTC),
	}

	mergeSHA, conflicts, err := Rebase(ctx, s, Params{
		Author:       committer,
		Committer:    committer,
		Message:      "this message must be ignored by rebase",
		MergeBaseSHA: baseSHA,
		TargetSHA:    targetSHA,
		SourceSHA:    sourceSHA,
	})
	require.NoError(t, err)
	require.Empty(t, conflicts)

	require.NoError(t, s.PackObjects(ctx))

	// the objects created by the rebase must have been combined into a single pack file
	packs, err := filepath.Glob(filepath.Join(s.Directory(), "objects", "pack", "*.pack"))
	require.NoError(t, err)
	require.Len(t, packs, 1)

	require.NoError(t, s.MoveObjects(ctx))

	// the rebased commits and everything they point to must be readable from the original
	// repository, without the shared repository (which is gone by then) as an alternate.
	assert.Empty(t, runGit(t, repoPath, "fsck", "--strict", "--no-dangling"))

	// the rebased branch consists of the 9 rebased commits on top of the base and the target.
	revs := strings.Fields(runGit(t, repoPath, "rev-list", mergeSHA.String()))
	assert.Len(t, revs, 11)
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}
}

// newTestRepo creates an empty non-bare git repository on the "main" branch.
func newTestRepo(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()
	runGit(t, repoPath, "init", "--initial-branch=main")

	return repoPath
}

// newSharedRepo creates a shared repository with the given repository as its object source.
func newSharedRepo(ctx context.Context, t *testing.T, repoPath string) *sharedrepo.SharedRepo {
	t.Helper()

	s, err := sharedrepo.NewSharedRepo(t.TempDir(), filepath.Join(repoPath, ".git"))
	require.NoError(t, err)
	t.Cleanup(func() { s.Close(ctx) })
	require.NoError(t, s.Init(ctx))

	return s
}

// commitFile writes a file and commits it with the message unmodified.
func commitFile(t *testing.T, repoPath, name, content, message string) sha.SHA {
	t.Helper()

	filePath := filepath.Join(repoPath, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o700))
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))
	runGit(t, repoPath, "add", name)

	messageFile := filepath.Join(t.TempDir(), "commit-message")
	require.NoError(t, os.WriteFile(messageFile, []byte(message), 0o600))
	runGit(t, repoPath, "commit", "--no-gpg-sign", "--cleanup=verbatim", "--file", messageFile)

	return sha.Must(runGit(t, repoPath, "rev-parse", "HEAD"))
}

func runGit(t *testing.T, repoPath string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test Author",
		"GIT_AUTHOR_EMAIL=author@test.io",
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00+00:00",
		"GIT_COMMITTER_NAME=Test Committer",
		"GIT_COMMITTER_EMAIL=committer@test.io",
		"GIT_COMMITTER_DATE=2020-01-02T00:00:00+00:00",
	)

	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s failed: %s", strings.Join(args, " "), out)

	return strings.TrimSpace(string(out))
}
