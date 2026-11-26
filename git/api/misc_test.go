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
	"testing"
	"time"

	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

func TestMapToReferenceSortingArgument(t *testing.T) {
	tests := []struct {
		name  string
		field GitReferenceField
		order SortOrder
		want  string
	}{
		{"default name asc", GitReferenceFieldRefName, SortOrderDefault, "refname"},
		{"name asc", GitReferenceFieldRefName, SortOrderAsc, "refname"},
		{"name desc", GitReferenceFieldRefName, SortOrderDesc, "-refname"},
		{"date default is desc", GitReferenceFieldCreatorDate, SortOrderDefault, "-creatordate"},
		{"date asc", GitReferenceFieldCreatorDate, SortOrderAsc, "creatordate"},
		{"date desc", GitReferenceFieldCreatorDate, SortOrderDesc, "-creatordate"},
		{"unknown field falls back to name", GitReferenceFieldObjectName, SortOrderAsc, "refname"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, mapToReferenceSortingArgument(tt.field, tt.order))
		})
	}
}

func TestMapRawRef(t *testing.T) {
	got, err := mapRawRef(map[string]string{"refname": "refs/heads/main", "objectname": "abc"})
	require.NoError(t, err)
	require.Equal(t, map[GitReferenceField]string{
		GitReferenceFieldRefName:    "refs/heads/main",
		GitReferenceFieldObjectName: "abc",
	}, got)

	_, err = mapRawRef(map[string]string{"bogus": "x"})
	require.Error(t, err)
}

func TestCommitEntryKey(t *testing.T) {
	commitSHA := sha.Must("1111111111111111111111111111111111111111")
	key := makeCommitEntryKey("/repo", commitSHA, "a/b.txt")

	repoPath, gotSHA, path := key.Split()
	require.Equal(t, "/repo", repoPath)
	require.Equal(t, commitSHA.String(), gotSHA)
	require.Equal(t, "a/b.txt", path)

	repoPath, gotSHA, path = CommitEntryKey("malformed").Split()
	require.Empty(t, repoPath)
	require.Empty(t, gotSHA)
	require.Empty(t, path)
}

func TestCommitValueCodec(t *testing.T) {
	codec := commitValueCodec{}
	in := &Commit{
		SHA:     sha.Must("1111111111111111111111111111111111111111"),
		Title:   "title",
		Message: "message",
		Author:  Signature{Identity: Identity{Name: "A", Email: "a@test.test"}, When: time.Unix(1700000000, 0).UTC()},
	}

	out, err := codec.Decode(codec.Encode(in))
	require.NoError(t, err)
	require.Equal(t, in.SHA, out.SHA)
	require.Equal(t, in.Title, out.Title)
	require.Equal(t, in.Message, out.Message)
	require.Equal(t, in.Author.Identity, out.Author.Identity)
	require.True(t, in.Author.When.Equal(out.Author.When))

	_, err = codec.Decode("not gob")
	require.Error(t, err)
}

func TestNewRedisLastCommitCache_NilClient(t *testing.T) {
	c, err := NewRedisLastCommitCache(nil, time.Minute)
	require.Error(t, err)
	require.Nil(t, c)
}

func TestLastCommitCachesAndPathsDetails(t *testing.T) {
	requireGit(t)
	repo := initRepoWithCommit(t)
	second := commitFile(t, repo, "dir/b.txt", "b\n", "second commit")

	for name, newCache := range map[string]func() *Git{
		"in memory": func() *Git { return &Git{lastCommitCache: NewInMemoryLastCommitCache(time.Minute)} },
		"no cache":  func() *Git { return &Git{lastCommitCache: NoLastCommitCache()} },
	} {
		t.Run(name, func(t *testing.T) {
			g := newCache()

			details, err := g.PathsDetails(t.Context(), repo, "main", []string{"a.txt", "dir/b.txt", "./dir"})
			require.NoError(t, err)
			require.Len(t, details, 3)
			require.Equal(t, "a.txt", details[0].Path)
			require.Equal(t, "first", details[0].LastCommit.Title)
			require.Equal(t, "dir/b.txt", details[1].Path)
			require.Equal(t, second, details[1].LastCommit.SHA.String())
			require.Equal(t, "./dir", details[2].Path)
			require.Equal(t, second, details[2].LastCommit.SHA.String())

			// repository root
			details, err = g.PathsDetails(t.Context(), repo, "main", []string{""})
			require.NoError(t, err)
			require.Equal(t, second, details[0].LastCommit.SHA.String())

			_, err = g.PathsDetails(t.Context(), repo, "no-such-rev", []string{"a.txt"})
			require.Error(t, err)
		})
	}
}

func TestPackRefs(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	runGit(t, repo, "branch", "feature")

	require.NoError(t, g.PackRefs(t.Context(), repo, PackRefsParams{All: true, NoPrune: true}))
	require.NoError(t, g.PackRefs(t.Context(), repo, PackRefsParams{Auto: true}))
	require.NoError(t, g.PackRefs(t.Context(), repo, PackRefsParams{
		All: true, Include: "refs/heads/*", Exclude: "refs/tags/*",
	}))
	require.Error(t, g.PackRefs(t.Context(), t.TempDir(), PackRefsParams{}))
}

func TestPackObjects(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)

	require.NoError(t, g.PackObjects(t.Context(), repo, PackObjectsParams{
		PackLooseUnreachable: true, Local: true, Incremental: true, NonEmpty: true, Quiet: true,
	}, ".git/objects/pack/test"))
	require.Error(t, g.PackObjects(t.Context(), t.TempDir(), PackObjectsParams{Quiet: true}, "pack"))
}
