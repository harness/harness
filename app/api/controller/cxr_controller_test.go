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

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errCxrBoom = errors.New("boom")

// cxrTransactor runs the function and records the options of every attempt.
type cxrTransactor struct {
	results []error
	calls   int
	opts    [][]any
}

func (t *cxrTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error, opts ...any) error {
	t.opts = append(t.opts, opts)
	var err error
	if fn != nil {
		err = fn(ctx)
	}
	if t.calls < len(t.results) && t.results[t.calls] != nil {
		err = t.results[t.calls]
	}
	t.calls++
	return err
}

func TestCxrTxOptLock(t *testing.T) {
	ctx := t.Context()
	noop := func(context.Context) error { return nil }

	t.Run("success runs once", func(t *testing.T) {
		tx := &cxrTransactor{}
		require.NoError(t, TxOptLock(ctx, tx, noop))
		require.Equal(t, 1, tx.calls)
	})

	t.Run("version conflicts are retried and reset", func(t *testing.T) {
		tx := &cxrTransactor{results: []error{store.ErrVersionConflict, store.ErrVersionConflict, nil}}
		resets := 0
		require.NoError(t, TxOptLock(ctx, tx, noop, TxOptionResetFunc(func() { resets++ })))
		require.Equal(t, 3, tx.calls)
		require.Equal(t, 2, resets)
	})

	t.Run("retry count limits the attempts", func(t *testing.T) {
		tx := &cxrTransactor{results: []error{
			store.ErrVersionConflict, store.ErrVersionConflict, store.ErrVersionConflict, store.ErrVersionConflict,
		}}
		err := TxOptLock(ctx, tx, noop, TxOptionRetryCount(2))
		require.ErrorIs(t, err, store.ErrVersionConflict)
		require.Equal(t, 2, tx.calls)
	})

	t.Run("default is five attempts", func(t *testing.T) {
		results := make([]error, 10)
		for i := range results {
			results[i] = store.ErrVersionConflict
		}
		tx := &cxrTransactor{results: results}
		require.ErrorIs(t, TxOptLock(ctx, tx, noop), store.ErrVersionConflict)
		require.Equal(t, 5, tx.calls)
	})

	t.Run("other errors are not retried", func(t *testing.T) {
		tx := &cxrTransactor{results: []error{errCxrBoom}}
		require.ErrorIs(t, TxOptLock(ctx, tx, noop), errCxrBoom)
		require.Equal(t, 1, tx.calls)
	})
}

type cxrURLProvider struct {
	url.Provider
}

func (cxrURLProvider) GetInternalAPIURL(context.Context) string { return "http://localhost:3000" }

func TestCxrCreateRPCWriteParams(t *testing.T) {
	ctx := t.Context()
	session := &auth.Session{Principal: types.Principal{
		ID: 7, DisplayName: "Jane", Email: "jane@example.com", Type: enum.PrincipalTypeUser,
	}}
	repo := &types.RepositoryCore{ID: 3, GitUID: "git-uid"}

	funcs := map[string]func(
		context.Context, url.Provider, *auth.Session, *types.RepositoryCore,
	) (git.WriteParams, error){
		"git push":     CreateRPCGitPushWriteParams,
		"api content":  CreateRPCAPIContentWriteParams,
		"bypass rules": CreateRPCAPIContentBypassRulesWriteParams,
		"api refs":     CreateRPCAPIRefsWriteParams,
		"system refs":  CreateRPCSystemReferencesWriteParams,
		"linked sync":  CreateRPCAPILinkedSyncWriteParams,
		"merge queue":  CreateRPCSystemMergeQueueWriteParams,
	}
	envs := map[string]map[string]string{}
	for name, fn := range funcs {
		params, err := fn(ctx, cxrURLProvider{}, session, repo)
		require.NoError(t, err, name)
		require.Equal(t, "git-uid", params.RepoUID, name)
		require.Equal(t, git.Identity{Name: "Jane", Email: "jane@example.com"}, params.Actor, name)
		require.NotEmpty(t, params.EnvVars, name)
		envs[name] = params.EnvVars
	}

	// The operation type is part of the payload, so every operation generates a different environment.
	seen := map[string]string{}
	for name, env := range envs {
		for _, v := range env {
			other, dup := seen[v]
			require.False(t, dup, "%s and %s share the same githook payload", name, other)
			seen[v] = name
		}
	}
}

func TestCxrMapCommit(t *testing.T) {
	require.Nil(t, MapCommit(nil))

	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	sig := git.Signature{Identity: git.Identity{Name: "Jane", Email: "jane@example.com"}, When: when}
	commit := &git.Commit{
		SHA:        sha.Must("1111111111111111111111111111111111111111"),
		TreeSHA:    sha.Must("2222222222222222222222222222222222222222"),
		ParentSHAs: []sha.SHA{sha.Must("3333333333333333333333333333333333333333")},
		Title:      "title",
		Message:    "message",
		Author:     sig,
		Committer:  sig,
		FileStats: []git.CommitFileStats{
			{Path: "a.go", Insertions: 3, Deletions: 1},
			{Path: "b.go", OldPath: "old.go", Insertions: 2, Deletions: 4},
		},
	}

	got := MapCommit(commit)
	require.Equal(t, commit.SHA, got.SHA)
	require.Equal(t, "title", got.Title)
	require.Equal(t, types.Signature{Identity: types.Identity{Name: "Jane", Email: "jane@example.com"}, When: when},
		got.Author)
	require.NotNil(t, got.Stats)
	require.EqualValues(t, 5, got.Stats.Total.Insertions)
	require.EqualValues(t, 5, got.Stats.Total.Deletions)
	require.EqualValues(t, 10, got.Stats.Total.Changes)
	require.Len(t, got.Stats.Files, 2)
	require.Equal(t, "old.go", got.Stats.Files[1].OldPath)
	require.EqualValues(t, 6, got.Stats.Files[1].Changes)

	// Commits without file statistics have no stats.
	commit.FileStats = nil
	require.Nil(t, MapCommit(commit).Stats)
}

func TestCxrMapBranchAndTag(t *testing.T) {
	branch, err := MapBranch(git.Branch{Name: "main", SHA: sha.Must("1111111111111111111111111111111111111111")})
	require.NoError(t, err)
	require.Equal(t, "main", branch.Name)
	require.Nil(t, branch.Commit)

	branch, err = MapBranch(git.Branch{Name: "dev", Commit: &git.Commit{Title: "t"}})
	require.NoError(t, err)
	require.Equal(t, "t", branch.Commit.Title)

	tag := MapCommitTag(git.CommitTag{Name: "v1", IsAnnotated: true, Title: "release", Message: "m"})
	require.Equal(t, "v1", tag.Name)
	require.True(t, tag.IsAnnotated)
	require.Nil(t, tag.Tagger)
	require.Nil(t, tag.Commit)

	tagger := git.Signature{Identity: git.Identity{Name: "Tagger", Email: "t@example.com"}}
	tag = MapCommitTag(git.CommitTag{Name: "v2", Tagger: &tagger, Commit: &git.Commit{Title: "c"}})
	require.Equal(t, "Tagger", tag.Tagger.Identity.Name)
	require.Equal(t, "c", tag.Commit.Title)
}

func TestCxrMapRenameDetailsAndIdentity(t *testing.T) {
	require.Nil(t, MapRenameDetails(nil))

	before := sha.Must("1111111111111111111111111111111111111111")
	after := sha.Must("2222222222222222222222222222222222222222")
	got := MapRenameDetails(&git.RenameDetails{
		OldPath: "a", NewPath: "b", CommitShaBefore: before, CommitShaAfter: after,
	})
	require.Equal(t, &types.RenameDetails{
		OldPath: "a", NewPath: "b", CommitShaBefore: before.String(), CommitShaAfter: after.String(),
	}, got)

	identity := IdentityFromPrincipalInfo(types.PrincipalInfo{DisplayName: "Jane", Email: "jane@example.com"})
	require.Equal(t, &git.Identity{Name: "Jane", Email: "jane@example.com"}, identity)
}
