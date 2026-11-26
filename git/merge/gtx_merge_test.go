// Copyright 2026 Harness, Inc.
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
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ Func = Merge
	_ Func = Squash
	_ Func = Rebase
	_ Func = FastForward
)

// gtxFixture is a bare repository with the following history:
//
//	base --- target (changes a.txt, adds t.txt)
//	  \------ clean1 --- clean2 (add c1.txt, c2.txt)
//	  \------ conflict (changes a.txt differently)
//	  \------ empty (an empty commit, then adds e.txt)
type gtxFixture struct {
	repo     string
	base     sha.SHA
	target   sha.SHA
	clean1   sha.SHA
	clean2   sha.SHA
	conflict sha.SHA
	emptyTip sha.SHA
}

func gtxGit(ctx context.Context, t *testing.T, dir string, args ...string) string {
	t.Helper()

	full := append([]string{
		"-c", "user.name=Gtx Tester",
		"-c", "user.email=gtx@example.com",
		"-c", "commit.gpgsign=false",
	}, args...)

	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), "git %v failed: %s", args, stderr.String())

	return strings.TrimSpace(stdout.String())
}

func gtxCommit(ctx context.Context, t *testing.T, dir, file, content, message string) sha.SHA {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600))
	gtxGit(ctx, t, dir, "add", file)
	gtxGit(ctx, t, dir, "commit", "-q", "-m", message)

	return sha.Must(gtxGit(ctx, t, dir, "rev-parse", "HEAD"))
}

func gtxNewFixture(ctx context.Context, t *testing.T) gtxFixture {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	work := t.TempDir()
	gtxGit(ctx, t, work, "init", "-q", "-b", "main")

	fx := gtxFixture{}
	fx.base = gtxCommit(ctx, t, work, "a.txt", "one\ntwo\nthree\n", "base")

	gtxGit(ctx, t, work, "checkout", "-q", "-b", "clean", fx.base.String())
	fx.clean1 = gtxCommit(ctx, t, work, "c1.txt", "c1\n", "clean one\n\nbody of clean one\n")
	fx.clean2 = gtxCommit(ctx, t, work, "c2.txt", "c2\n", "clean two")

	gtxGit(ctx, t, work, "checkout", "-q", "-b", "conflict", fx.base.String())
	fx.conflict = gtxCommit(ctx, t, work, "a.txt", "one\nTWO-conflict\nthree\n", "conflicting change")

	gtxGit(ctx, t, work, "checkout", "-q", "-b", "empty", fx.base.String())
	gtxGit(ctx, t, work, "commit", "-q", "--allow-empty", "-m", "empty commit")
	fx.emptyTip = gtxCommit(ctx, t, work, "e.txt", "e\n", "adds e")

	gtxGit(ctx, t, work, "checkout", "-q", "main")
	require.NoError(t, os.WriteFile(filepath.Join(work, "t.txt"), []byte("t\n"), 0o600))
	gtxGit(ctx, t, work, "add", "t.txt")
	fx.target = gtxCommit(ctx, t, work, "a.txt", "one\nTWO-target\nthree\n", "target change")

	fx.repo = filepath.Join(t.TempDir(), "repo.git")
	gtxGit(ctx, t, work, "clone", "-q", "--bare", work, fx.repo)

	return fx
}

func gtxNewShared(ctx context.Context, t *testing.T, repo string) *sharedrepo.SharedRepo {
	t.Helper()

	s, err := sharedrepo.NewSharedRepo(t.TempDir(), repo)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close(ctx) })
	require.NoError(t, s.Init(ctx))

	return s
}

func gtxSignature(name string) *api.Signature {
	return &api.Signature{
		Identity: api.Identity{Name: name, Email: strings.ToLower(name) + "@example.com"},
		When:     time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC),
	}
}

func gtxTreeFiles(ctx context.Context, t *testing.T, s *sharedrepo.SharedRepo, rev string) []string {
	t.Helper()

	return strings.Split(gtxGit(ctx, t, s.Directory(), "ls-tree", "-r", "--name-only", rev), "\n")
}

func TestGtxMergeAndSquash(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)

	tests := []struct {
		name        string
		fn          Func
		source      sha.SHA
		wantParents int
	}{
		{name: "merge", fn: Merge, source: fx.clean2, wantParents: 2},
		{name: "squash", fn: Squash, source: fx.clean2, wantParents: 1},
	}

	for _, test := range tests {
		t.Run(test.name+" without conflicts", func(t *testing.T) {
			s := gtxNewShared(ctx, t, fx.repo)

			mergeSHA, conflicts, err := test.fn(ctx, s, Params{
				Author:       gtxSignature("Author"),
				Committer:    gtxSignature("Committer"),
				Message:      "gtx merge message",
				MergeBaseSHA: fx.base,
				TargetSHA:    fx.target,
				SourceSHA:    test.source,
			})
			require.NoError(t, err)
			assert.Empty(t, conflicts)
			assert.False(t, mergeSHA.IsEmpty())

			commit, err := api.GetCommit(ctx, s.Directory(), mergeSHA)
			require.NoError(t, err)
			assert.Len(t, commit.ParentSHAs, test.wantParents)
			assert.Equal(t, fx.target, commit.ParentSHAs[0])
			assert.Equal(t, "gtx merge message", commit.Title)
			assert.Equal(t, "Author", commit.Author.Identity.Name)
			assert.Equal(t, "Committer", commit.Committer.Identity.Name)
			assert.ElementsMatch(t, []string{"a.txt", "c1.txt", "c2.txt", "t.txt"}, gtxTreeFiles(ctx, t, s, mergeSHA.String()))
		})

		t.Run(test.name+" with conflicts", func(t *testing.T) {
			s := gtxNewShared(ctx, t, fx.repo)

			mergeSHA, conflicts, err := test.fn(ctx, s, Params{
				Author:       gtxSignature("Author"),
				Committer:    gtxSignature("Committer"),
				Message:      "gtx merge message",
				MergeBaseSHA: fx.base,
				TargetSHA:    fx.target,
				SourceSHA:    fx.conflict,
			})
			require.NoError(t, err)
			assert.Equal(t, []string{"a.txt"}, conflicts)
			assert.True(t, mergeSHA.IsEmpty())
		})

		t.Run(test.name+" with unknown commit", func(t *testing.T) {
			s := gtxNewShared(ctx, t, fx.repo)

			_, _, err := test.fn(ctx, s, Params{
				Author:       gtxSignature("Author"),
				Committer:    gtxSignature("Committer"),
				MergeBaseSHA: sha.None,
				TargetSHA:    fx.target,
				SourceSHA:    sha.Must("1234567890123456789012345678901234567890"),
			})
			require.ErrorContains(t, err, "merge tree failed")
		})
	}
}

func TestGtxRebase(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)

	committer := gtxSignature("Committer")

	t.Run("multiple commits are replayed in order", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.repo)

		mergeSHA, conflicts, err := Rebase(ctx, s, Params{
			Committer: committer, MergeBaseSHA: fx.base, TargetSHA: fx.target, SourceSHA: fx.clean2,
		})
		require.NoError(t, err)
		assert.Empty(t, conflicts)

		log := gtxGit(ctx, t, s.Directory(), "log", "--format=%s", "--first-parent", mergeSHA.String())
		assert.Equal(t, []string{"clean two", "clean one", "target change", "base"}, strings.Split(log, "\n"))
		assert.ElementsMatch(t, []string{"a.txt", "c1.txt", "c2.txt", "t.txt"}, gtxTreeFiles(ctx, t, s, mergeSHA.String()))

		top, err := api.GetCommit(ctx, s.Directory(), mergeSHA)
		require.NoError(t, err)
		assert.Equal(t, "Committer", top.Committer.Identity.Name)
		assert.Equal(t, "Gtx Tester", top.Author.Identity.Name, "author is preserved from the source commit")
	})

	t.Run("nothing to rebase returns the target", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.repo)

		mergeSHA, conflicts, err := Rebase(ctx, s, Params{
			Committer: committer, MergeBaseSHA: fx.base, TargetSHA: fx.target, SourceSHA: fx.base,
		})
		require.NoError(t, err)
		assert.Empty(t, conflicts)
		assert.Equal(t, fx.target, mergeSHA)
	})

	t.Run("empty commits are dropped", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.repo)

		mergeSHA, conflicts, err := Rebase(ctx, s, Params{
			Committer: committer, MergeBaseSHA: fx.base, TargetSHA: fx.target, SourceSHA: fx.emptyTip,
		})
		require.NoError(t, err)
		assert.Empty(t, conflicts)

		log := gtxGit(ctx, t, s.Directory(), "log", "--format=%s", "--first-parent", mergeSHA.String())
		assert.Equal(t, []string{"adds e", "target change", "base"}, strings.Split(log, "\n"))
	})

	t.Run("conflicts are reported", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.repo)

		mergeSHA, conflicts, err := Rebase(ctx, s, Params{
			Committer: committer, MergeBaseSHA: fx.base, TargetSHA: fx.target, SourceSHA: fx.conflict,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"a.txt"}, conflicts)
		assert.True(t, mergeSHA.IsEmpty())
	})

	t.Run("unknown source commit", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.repo)

		_, _, err := Rebase(ctx, s, Params{
			Committer: committer, MergeBaseSHA: fx.base, TargetSHA: fx.target,
			SourceSHA: sha.Must("1234567890123456789012345678901234567890"),
		})
		require.ErrorContains(t, err, "failed to find commit list in rebase merge")
	})

	t.Run("unknown target commit", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.repo)

		_, _, err := Rebase(ctx, s, Params{
			Committer: committer, MergeBaseSHA: fx.base, SourceSHA: fx.clean1,
			TargetSHA: sha.Must("1234567890123456789012345678901234567890"),
		})
		require.Error(t, err)
	})
}

func TestGtxFastForward(t *testing.T) {
	ctx := context.Background()

	base := sha.Must("1111111111111111111111111111111111111111")
	source := sha.Must("2222222222222222222222222222222222222222")
	other := sha.Must("3333333333333333333333333333333333333333")

	t.Run("target is the merge base", func(t *testing.T) {
		got, conflicts, err := FastForward(ctx, nil, Params{MergeBaseSHA: base, TargetSHA: base, SourceSHA: source})
		require.NoError(t, err)
		assert.Empty(t, conflicts)
		assert.Equal(t, source, got)
	})

	t.Run("diverged target", func(t *testing.T) {
		got, conflicts, err := FastForward(ctx, nil, Params{MergeBaseSHA: base, TargetSHA: other, SourceSHA: source})
		require.Error(t, err)
		assert.True(t, errors.IsConflict(err), "unexpected error: %v", err)
		assert.Empty(t, conflicts)
		assert.True(t, got.IsEmpty())
	})
}

func TestGtxFindConflicts(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)

	tests := []struct {
		name          string
		base, head    string
		wantMergeable bool
		wantConflicts []string
	}{
		{name: "clean", base: fx.target.String(), head: fx.clean2.String(), wantMergeable: true},
		{name: "conflict", base: fx.target.String(), head: fx.conflict.String(), wantConflicts: []string{"a.txt"}},
		{name: "branch names", base: "main", head: "clean", wantMergeable: true},
		{name: "same revision", base: "main", head: "main", wantMergeable: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mergeable, tree, conflicts, err := FindConflicts(ctx, fx.repo, test.base, test.head)
			require.NoError(t, err)
			assert.Equal(t, test.wantMergeable, mergeable)
			assert.Equal(t, test.wantConflicts, conflicts)
			assert.Len(t, tree, 40)
		})
	}

	t.Run("unknown revision", func(t *testing.T) {
		mergeable, tree, conflicts, err := FindConflicts(ctx, fx.repo, "main", "gtx-unknown")
		require.Error(t, err)
		assert.True(t, errors.AsStatus(err) == errors.StatusInternal, "unexpected error: %v", err)
		assert.False(t, mergeable)
		assert.Empty(t, tree)
		assert.Empty(t, conflicts)
	})

	t.Run("missing repository", func(t *testing.T) {
		_, _, _, err := FindConflicts(ctx, filepath.Join(t.TempDir(), "gone"), "main", "clean")
		require.Error(t, err)
	})
}

func TestGtxCommitCount(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)

	tests := []struct {
		name       string
		start, end string
		want       int
	}{
		{name: "whole history of main", end: "main", want: 2},
		{name: "range", start: "main", end: "clean", want: 2},
		{name: "empty range", start: "clean", end: "clean", want: 0},
		{name: "by sha", start: fx.base.String(), end: fx.emptyTip.String(), want: 2},
		{name: "reversed range", start: "clean", end: "main", want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := CommitCount(ctx, fx.repo, test.start, test.end)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}

	t.Run("unknown revision", func(t *testing.T) {
		_, err := CommitCount(ctx, fx.repo, "", "gtx-unknown")
		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err), "unexpected error: %v", err)
	})

	t.Run("missing repository", func(t *testing.T) {
		_, err := CommitCount(ctx, filepath.Join(t.TempDir(), "gone"), "", "main")
		require.Error(t, err)
		assert.False(t, errors.IsNotFound(err), "unexpected error: %v", err)
	})
}
