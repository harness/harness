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
	"testing"

	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

func TestGetMergeBase(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	base := runGit(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "checkout", "-q", "-b", "feature")
	commitFile(t, repo, "f.txt", "f\n", "feature commit")
	runGit(t, repo, "checkout", "-q", "main")
	commitFile(t, repo, "m.txt", "m\n", "main commit")

	_, _, err := g.GetMergeBase(t.Context(), "", "", "main", "feature", false)
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	t.Run("simple", func(t *testing.T) {
		got, usedBase, err := g.GetMergeBase(t.Context(), repo, "", "main", "feature", false)
		require.NoError(t, err)
		require.Equal(t, base, got.String())
		require.Equal(t, "main", usedBase)
	})

	t.Run("custom remote", func(t *testing.T) {
		runGit(t, repo, "remote", "add", "upstream", repo)
		got, _, err := g.GetMergeBase(t.Context(), repo, "upstream", "main", "feature", true)
		require.NoError(t, err)
		require.Equal(t, base, got.String())
	})

	t.Run("unknown revision", func(t *testing.T) {
		_, _, err := g.GetMergeBase(t.Context(), repo, "", "main", "missing", false)
		require.Error(t, err)
	})

	t.Run("unrelated histories", func(t *testing.T) {
		runGit(t, repo, "checkout", "-q", "--orphan", "orphan")
		runGit(t, repo, "rm", "-rfq", ".")
		commitFile(t, repo, "o.txt", "o\n", "orphan commit")
		runGit(t, repo, "checkout", "-q", "main")

		_, _, err := g.GetMergeBase(t.Context(), repo, "", "main", "orphan", false)
		require.True(t, IsUnrelatedHistoriesError(err), "got %v", err)
	})
}

func TestGetMergeBase_MultipleMergeBases(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)

	// criss-cross merge produces two merge bases
	runGit(t, repo, "checkout", "-q", "-b", "left")
	commitFile(t, repo, "l.txt", "l\n", "left")
	runGit(t, repo, "checkout", "-q", "main")
	runGit(t, repo, "checkout", "-q", "-b", "right")
	commitFile(t, repo, "r.txt", "r\n", "right")
	runGit(t, repo, "checkout", "-q", "-b", "left2", "left")
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "merge right into left", "right")
	runGit(t, repo, "checkout", "-q", "right")
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "merge left into right", "left")

	_, _, err := g.GetMergeBase(t.Context(), repo, "", "left2", "right", false)
	require.Error(t, err)
	require.True(t, IsMergeBaseNonUniqueError(err) || errors.Is(err, ErrMergeBaseNonUnique), "got %v", err)

	got, _, err := g.GetMergeBase(t.Context(), repo, "", "left2", "right", true)
	require.NoError(t, err)
	require.False(t, got.IsEmpty())
}

func TestIsAncestor(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	first := sha.Must(runGit(t, repo, "rev-parse", "HEAD"))
	second := sha.Must(commitFile(t, repo, "b.txt", "b\n", "second"))

	_, err := g.IsAncestor(t.Context(), "", nil, first, second)
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	ok, err := g.IsAncestor(t.Context(), repo, nil, first, second)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = g.IsAncestor(t.Context(), repo, nil, second, first)
	require.NoError(t, err)
	require.False(t, ok)

	_, err = g.IsAncestor(t.Context(), repo, nil, sha.Must("1111111111111111111111111111111111111111"), second)
	require.Error(t, err)
}
