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
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInitRepository(t *testing.T) {
	requireGit(t)
	g := &Git{}

	require.ErrorIs(t, g.InitRepository(t.Context(), "", false), ErrRepositoryPathEmpty)

	normal := filepath.Join(t.TempDir(), "nested", "normal")
	require.NoError(t, g.InitRepository(t.Context(), normal, false))
	require.DirExists(t, filepath.Join(normal, ".git"))

	bare := filepath.Join(t.TempDir(), "bare.git")
	require.NoError(t, g.InitRepository(t.Context(), bare, true))
	require.FileExists(t, filepath.Join(bare, "HEAD"))
}

func TestDefaultBranch(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	runGit(t, repo, "branch", "develop")

	got, err := g.GetDefaultBranch(t.Context(), repo)
	require.NoError(t, err)
	require.Equal(t, "main", got)

	raw, err := g.GetSymbolicRefHeadRaw(t.Context(), repo)
	require.NoError(t, err)
	require.Equal(t, "refs/heads/main", raw[:len("refs/heads/main")])

	_, err = g.GetSymbolicRefHeadRaw(t.Context(), "")
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)
	_, err = g.GetDefaultBranch(t.Context(), "")
	require.Error(t, err)

	require.ErrorIs(t, g.SetDefaultBranch(t.Context(), "", "develop", false), ErrRepositoryPathEmpty)
	require.Error(t, g.SetDefaultBranch(t.Context(), repo, "missing", false))

	require.NoError(t, g.SetDefaultBranch(t.Context(), repo, "develop", false))
	got, err = g.GetDefaultBranch(t.Context(), repo)
	require.NoError(t, err)
	require.Equal(t, "develop", got)

	// ignoring existence allows pointing HEAD to an unborn branch
	require.NoError(t, g.SetDefaultBranch(t.Context(), repo, "unborn", true))
	got, err = g.GetDefaultBranch(t.Context(), repo)
	require.NoError(t, err)
	require.Equal(t, "unborn", got)
}

func TestGetRemoteDefaultBranch(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)

	got, err := g.GetRemoteDefaultBranch(t.Context(), repo)
	require.NoError(t, err)
	require.Equal(t, "main", got)

	empty := t.TempDir()
	runGit(t, empty, "init", "--bare", "--initial-branch", "main", ".")
	_, err = g.GetRemoteDefaultBranch(t.Context(), empty)
	require.ErrorIs(t, err, ErrNoDefaultBranch)

	_, err = g.GetRemoteDefaultBranch(t.Context(), filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func TestCloneAndSync(t *testing.T) {
	requireGit(t)
	g := &Git{}
	origin := initRepoWithCommit(t)
	runGit(t, origin, "branch", "feature")

	t.Run("clone variants", func(t *testing.T) {
		full := filepath.Join(t.TempDir(), "full")
		require.NoError(t, g.Clone(t.Context(), origin, full, CloneRepoOptions{Quiet: true, Branch: "feature"}))
		require.Equal(t, "feature", runGit(t, full, "rev-parse", "--abbrev-ref", "HEAD"))

		bare := filepath.Join(t.TempDir(), "bare")
		require.NoError(t, g.Clone(t.Context(), origin, bare, CloneRepoOptions{Bare: true, Quiet: true}))
		require.FileExists(t, filepath.Join(bare, "HEAD"))

		mirror := filepath.Join(t.TempDir(), "mirror")
		require.NoError(t, g.Clone(t.Context(), origin, mirror, CloneRepoOptions{Mirror: true, Quiet: true}))
		require.FileExists(t, filepath.Join(mirror, "HEAD"))

		noCheckout := filepath.Join(t.TempDir(), "nocheckout")
		require.NoError(t, g.Clone(t.Context(), origin, noCheckout,
			CloneRepoOptions{NoCheckout: true, Shared: true, Quiet: true}))
		require.NoFileExists(t, filepath.Join(noCheckout, "a.txt"))

		shallow := filepath.Join(t.TempDir(), "shallow")
		require.NoError(t, g.Clone(t.Context(), "file://"+origin, shallow,
			CloneRepoOptions{Depth: 1, Quiet: true, SkipTLSVerify: true}))
		require.FileExists(t, filepath.Join(shallow, ".git", "shallow"))

		filtered := filepath.Join(t.TempDir(), "filtered")
		require.NoError(t, g.Clone(t.Context(), "file://"+origin, filtered,
			CloneRepoOptions{Filter: "blob:none", Quiet: true}))
	})

	t.Run("clone failure", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing")
		err := g.Clone(t.Context(), missing, filepath.Join(t.TempDir(), "to"), CloneRepoOptions{})
		require.Error(t, err)
	})

	t.Run("sync", func(t *testing.T) {
		mirror := filepath.Join(t.TempDir(), "mirror.git")
		require.NoError(t, g.InitRepository(t.Context(), mirror, true))

		require.ErrorIs(t, g.Sync(t.Context(), "", origin, nil), ErrRepositoryPathEmpty)
		require.NoError(t, g.Sync(t.Context(), mirror, origin, nil))

		got, err := g.ListLocalReferences(t.Context(), mirror)
		require.NoError(t, err)
		require.Contains(t, got, "refs/heads/main")
		require.Contains(t, got, "refs/heads/feature")

		runGit(t, origin, "branch", "-D", "feature")
		require.NoError(t, g.Sync(t.Context(), mirror, origin, []string{"+refs/heads/*:refs/heads/*"}))
		got, err = g.ListLocalReferences(t.Context(), mirror)
		require.NoError(t, err)
		require.NotContains(t, got, "refs/heads/feature")

		require.Error(t, g.Sync(t.Context(), mirror, filepath.Join(t.TempDir(), "missing"), nil))
	})
}

func TestListReferences(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	runGit(t, repo, "tag", "v1")
	head := runGit(t, repo, "rev-parse", "HEAD")

	remote, err := g.ListRemoteReferences(t.Context(), repo, repo)
	require.NoError(t, err)
	require.Equal(t, head, remote["refs/heads/main"].String())
	require.Equal(t, head, remote["refs/tags/v1"].String())

	filtered, err := g.ListRemoteReferences(t.Context(), repo, repo, "refs/tags/*")
	require.NoError(t, err)
	require.Len(t, filtered, 1)

	_, err = g.ListRemoteReferences(t.Context(), "", repo)
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)
	_, err = g.ListRemoteReferences(t.Context(), repo, filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)

	local, err := g.ListLocalReferences(t.Context(), repo)
	require.NoError(t, err)
	require.Len(t, local, 2)

	one, err := g.ListLocalReferences(t.Context(), repo, "refs/tags/v1")
	require.NoError(t, err)
	require.Len(t, one, 1)
	require.Equal(t, head, one["refs/tags/v1"].String())
}

func TestAddFilesAndCommit(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)

	require.ErrorIs(t, g.AddFiles(t.Context(), "", true), ErrRepositoryPathEmpty)
	require.ErrorIs(t, g.Commit(t.Context(), "", CommitChangesOptions{}), ErrRepositoryPathEmpty)

	signature := Signature{
		Identity: Identity{Name: "Committer", Email: "committer@test.test"},
		When:     time.Unix(1700000000, 0).UTC(),
	}
	opts := CommitChangesOptions{Message: "via api", Author: signature, Committer: signature}

	writeRepoFile(t, repo, "x.txt", "x\n")
	writeRepoFile(t, repo, "y.txt", "y\n")
	require.NoError(t, g.AddFiles(t.Context(), repo, false, "x.txt"))
	require.NoError(t, g.Commit(t.Context(), repo, opts))
	require.Equal(t, "via api", runGit(t, repo, "log", "-1", "--format=%s"))
	require.Equal(t, "Committer", runGit(t, repo, "log", "-1", "--format=%an"))
	require.Equal(t, "x.txt", runGit(t, repo, "show", "--name-only", "--format=", "HEAD"))

	require.NoError(t, g.AddFiles(t.Context(), repo, true))
	opts.Message = "second via api"
	require.NoError(t, g.Commit(t.Context(), repo, opts))
	require.Equal(t, "y.txt", runGit(t, repo, "show", "--name-only", "--format=", "HEAD"))

	// nothing to commit is not an error
	require.NoError(t, g.Commit(t.Context(), repo, opts))
	require.Equal(t, "second via api", runGit(t, repo, "log", "-1", "--format=%s"))

	require.Error(t, g.AddFiles(t.Context(), repo, false, "does-not-exist.txt"))
}

func TestPush(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, t.TempDir(), "init", "--bare", "--initial-branch", "main", remote)

	require.ErrorIs(t, g.Push(t.Context(), "", PushOptions{}), ErrRepositoryPathEmpty)

	require.NoError(t, g.Push(t.Context(), repo, PushOptions{Remote: remote, Branch: "main"}))
	require.Equal(t, runGit(t, repo, "rev-parse", "HEAD"), runGit(t, remote, "rev-parse", "main"))

	t.Run("non fast forward", func(t *testing.T) {
		other := filepath.Join(t.TempDir(), "other")
		require.NoError(t, g.Clone(t.Context(), remote, other, CloneRepoOptions{Quiet: true}))
		commitFile(t, other, "other.txt", "o\n", "from other")
		require.NoError(t, g.Push(t.Context(), other, PushOptions{Remote: remote, Branch: "main"}))

		runGit(t, repo, "fetch", remote, "main")
		commitFile(t, repo, "mine.txt", "m\n", "diverging")
		err := g.Push(t.Context(), repo, PushOptions{Remote: remote, Branch: "main"})
		var outOfDate *PushOutOfDateError
		require.True(t, errors.As(err, &outOfDate), "got %v", err)

		require.NoError(t, g.Push(t.Context(), repo, PushOptions{Remote: remote, Branch: "main", Force: true}))
		require.Equal(t, runGit(t, repo, "rev-parse", "HEAD"), runGit(t, remote, "rev-parse", "main"))
	})

	t.Run("force with lease and mirror", func(t *testing.T) {
		remoteHead := runGit(t, remote, "rev-parse", "main")
		commitFile(t, repo, "lease.txt", "l\n", "lease")
		require.NoError(t, g.Push(t.Context(), repo, PushOptions{
			Remote: remote, Branch: "main", ForceWithLease: "main:" + remoteHead,
		}))
		require.NoError(t, g.Push(t.Context(), repo, PushOptions{Remote: remote, Mirror: true}))
	})

	t.Run("rejected by hook", func(t *testing.T) {
		hook := filepath.Join(remote, "hooks", "pre-receive")
		require.NoError(t, os.WriteFile(hook, []byte("#!/bin/sh\necho 'policy says no' >&2\nexit 1\n"), 0o755)) //nolint:gosec
		defer func() { _ = os.Remove(hook) }()
		commitFile(t, repo, "rejected.txt", "r\n", "rejected")
		err := g.Push(t.Context(), repo, PushOptions{Remote: remote, Branch: "main"})
		require.True(t, IsErrPushRejected(err), "got %v", err)
		var rejected *PushRejectedError
		require.True(t, errors.As(err, &rejected))
		require.Equal(t, "policy says no", rejected.Message)
	})

	t.Run("ambiguous ref", func(t *testing.T) {
		runGit(t, repo, "tag", "ambiguous")
		runGit(t, repo, "branch", "ambiguous")
		err := g.Push(t.Context(), repo, PushOptions{Remote: remote, Branch: "ambiguous"})
		require.True(t, IsErrMoreThanOne(err), "got %v", err)
	})

	t.Run("unknown remote", func(t *testing.T) {
		require.Error(t, g.Push(t.Context(), repo, PushOptions{
			Remote: "https://user:secret@127.0.0.1:1/missing.git", Branch: "main", Env: []string{"A=B"},
		}))
	})

	t.Run("trace enabled", func(t *testing.T) {
		traced := &Git{traceGit: true}
		require.NoError(t, traced.Push(t.Context(), repo, PushOptions{Remote: remote, Branch: "main:refs/heads/traced"}))
	})
}

func TestCountObjects(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)

	got, err := g.CountObjects(t.Context(), repo)
	require.NoError(t, err)
	require.Positive(t, got.Count)

	_, err = g.CountObjects(t.Context(), t.TempDir())
	require.Error(t, err)
}

func TestParseGitCountObjectsOutput(t *testing.T) {
	out := `count: 1
size: 2
in-pack: 3
packs: 4
size-pack: 5
prune-packable: 6
garbage: 7
size-garbage: 8
alternate: /somewhere
`
	require.Equal(t, ObjectCount{
		Count: 1, Size: 2, InPack: 3, Packs: 4, SizePack: 5, PrunePackable: 6, Garbage: 7, SizeGarbage: 8,
	}, parseGitCountObjectsOutput(t.Context(), out))
}
