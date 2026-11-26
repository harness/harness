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

	"github.com/stretchr/testify/require"
)

func TestEnsureBranchPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"main", "refs/heads/main"},
		{"refs/heads/main", "refs/heads/main"},
		{"feature/x", "refs/heads/feature/x"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			require.Equal(t, tt.want, EnsureBranchPrefix(tt.in))
		})
	}
}

func TestGetReferenceFromNames(t *testing.T) {
	require.Equal(t, "refs/heads/main", GetReferenceFromBranchName("  main "))
	require.Equal(t, "refs/heads/main", GetReferenceFromBranchName("refs/heads/main"))
	require.Equal(t, "refs/tags/v1", GetReferenceFromTagName(" v1"))
	require.Equal(t, "refs/tags/v1", GetReferenceFromTagName("refs/tags/v1"))
}

func TestParseGitReferenceField(t *testing.T) {
	for _, f := range []GitReferenceField{
		GitReferenceFieldRefName, GitReferenceFieldObjectName,
		GitReferenceFieldObjectType, GitReferenceFieldCreatorDate,
	} {
		got, err := ParseGitReferenceField(string(f))
		require.NoError(t, err)
		require.Equal(t, f, got)
	}

	got, err := ParseGitReferenceField("nope")
	require.Error(t, err)
	require.Equal(t, GitReferenceFieldRefName, got)
}

func TestParseGitObjectType(t *testing.T) {
	for _, o := range []GitObjectType{
		GitObjectTypeCommit, GitObjectTypeTree, GitObjectTypeBlob, GitObjectTypeTag,
	} {
		got, err := ParseGitObjectType(string(o))
		require.NoError(t, err)
		require.Equal(t, o, got)
	}

	_, err := ParseGitObjectType("nope")
	require.Error(t, err)
}

func TestGetBranch(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	head := runGit(t, repo, "rev-parse", "HEAD")

	t.Run("empty repo path", func(t *testing.T) {
		_, err := g.GetBranch(t.Context(), "", "main")
		require.True(t, errors.Is(err, ErrRepositoryPathEmpty))
	})
	t.Run("empty branch name", func(t *testing.T) {
		_, err := g.GetBranch(t.Context(), repo, "")
		require.True(t, errors.Is(err, ErrBranchNameEmpty))
	})
	t.Run("existing branch", func(t *testing.T) {
		b, err := g.GetBranch(t.Context(), repo, "main")
		require.NoError(t, err)
		require.Equal(t, "main", b.Name)
		require.Equal(t, head, b.SHA.String())
		require.NotNil(t, b.Commit)
	})
	t.Run("missing branch", func(t *testing.T) {
		_, err := g.GetBranch(t.Context(), repo, "does-not-exist")
		require.Error(t, err)
	})
}

func TestIsBranchExistAndCount(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	runGit(t, repo, "branch", "feature")

	ok, err := g.IsBranchExist(t.Context(), repo, "feature")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = g.IsBranchExist(t.Context(), repo, "nope")
	require.NoError(t, err)
	require.False(t, ok)

	n, err := g.GetBranchCount(t.Context(), repo)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	_, err = g.GetBranchCount(t.Context(), "")
	require.True(t, errors.Is(err, ErrRepositoryPathEmpty))
}

func TestHasBranches(t *testing.T) {
	requireGit(t)
	g := &Git{}

	_, err := g.HasBranches(t.Context(), "")
	require.True(t, errors.Is(err, ErrRepositoryPathEmpty))

	// NOTE: the implementation reports "true" when rev-list returns no commit.
	empty := t.TempDir()
	runGit(t, empty, "init", "--initial-branch", "main", ".")
	got, err := g.HasBranches(t.Context(), empty)
	require.NoError(t, err)
	require.True(t, got)

	repo := initRepoWithCommit(t)
	got, err = g.HasBranches(t.Context(), repo)
	require.NoError(t, err)
	require.False(t, got)
}

func TestGetRef(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	head := runGit(t, repo, "rev-parse", "HEAD")

	_, err := g.GetRef(t.Context(), "", "refs/heads/main")
	require.True(t, errors.Is(err, ErrRepositoryPathEmpty))

	got, err := g.GetRef(t.Context(), repo, "refs/heads/main")
	require.NoError(t, err)
	require.Equal(t, head, got.String())

	_, err = g.GetRef(t.Context(), repo, "refs/heads/missing")
	require.Error(t, err)
}

func TestWalkReferences(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	runGit(t, repo, "branch", "b1")
	runGit(t, repo, "branch", "b2")
	runGit(t, repo, "tag", "t1")

	collect := func(opts *WalkReferencesOptions) []string {
		var names []string
		err := g.WalkReferences(t.Context(), repo, func(e WalkReferencesEntry) error {
			names = append(names, e[GitReferenceFieldRefName])
			return nil
		}, opts)
		require.NoError(t, err)
		return names
	}

	t.Run("empty repo path", func(t *testing.T) {
		err := g.WalkReferences(t.Context(), "", nil, &WalkReferencesOptions{})
		require.True(t, errors.Is(err, ErrRepositoryPathEmpty))
	})
	t.Run("all references sorted", func(t *testing.T) {
		require.Equal(t,
			[]string{"refs/heads/b1", "refs/heads/b2", "refs/heads/main", "refs/tags/t1"},
			collect(&WalkReferencesOptions{}))
	})
	t.Run("pattern and descending order", func(t *testing.T) {
		require.Equal(t,
			[]string{"refs/heads/main", "refs/heads/b2", "refs/heads/b1"},
			collect(&WalkReferencesOptions{Patterns: []string{"refs/heads"}, Order: SortOrderDesc}))
	})
	t.Run("max walk distance", func(t *testing.T) {
		require.Len(t, collect(&WalkReferencesOptions{MaxWalkDistance: 2}), 2)
	})
	t.Run("instructor skip and stop", func(t *testing.T) {
		calls := 0
		opts := &WalkReferencesOptions{
			Instructor: func(e WalkReferencesEntry) (WalkInstruction, error) {
				calls++
				switch e[GitReferenceFieldRefName] {
				case "refs/heads/b1":
					return WalkInstructionSkip, nil
				case "refs/heads/main":
					return WalkInstructionStop, nil
				default:
					return WalkInstructionHandle, nil
				}
			},
		}
		require.Equal(t, []string{"refs/heads/b2"}, collect(opts))
		require.Equal(t, 3, calls)
	})
	t.Run("instructor error", func(t *testing.T) {
		err := g.WalkReferences(t.Context(), repo, func(WalkReferencesEntry) error { return nil },
			&WalkReferencesOptions{
				Instructor: func(WalkReferencesEntry) (WalkInstruction, error) {
					return WalkInstructionStop, errors.New("boom")
				},
			})
		require.Error(t, err)
	})
	t.Run("handler error", func(t *testing.T) {
		err := g.WalkReferences(t.Context(), repo, func(WalkReferencesEntry) error { return errors.New("boom") },
			&WalkReferencesOptions{})
		require.Error(t, err)
	})
	t.Run("extra fields", func(t *testing.T) {
		var types []string
		err := g.WalkReferences(t.Context(), repo, func(e WalkReferencesEntry) error {
			types = append(types, e[GitReferenceFieldObjectType])
			return nil
		}, &WalkReferencesOptions{
			Fields:   []GitReferenceField{GitReferenceFieldRefName, GitReferenceFieldObjectType},
			Patterns: []string{"refs/tags"},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"commit"}, types)
	})
}

func TestResolveRev(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	head := runGit(t, repo, "rev-parse", "HEAD")

	got, err := g.ResolveRev(t.Context(), repo, "main")
	require.NoError(t, err)
	require.Equal(t, head, got.String())

	_, err = g.ResolveRev(t.Context(), repo, "does-not-exist")
	require.Error(t, err)
}
