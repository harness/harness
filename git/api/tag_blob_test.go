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
	"io"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

func TestCreateTag(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	head := sha.Must(runGit(t, repo, "rev-parse", "HEAD"))

	require.True(t, errors.Is(g.CreateTag(t.Context(), "", "v0", head, nil), ErrRepositoryPathEmpty))

	t.Run("lightweight", func(t *testing.T) {
		require.NoError(t, g.CreateTag(t.Context(), repo, "light", head, nil))
		require.Equal(t, "commit", runGit(t, repo, "cat-file", "-t", "refs/tags/light"))
	})

	t.Run("annotated", func(t *testing.T) {
		require.NoError(t, g.CreateTag(t.Context(), repo, "annotated", head, &CreateTagOptions{
			Message: "release one",
			Tagger: Signature{
				Identity: Identity{Name: "Tagger", Email: "tagger@test.test"},
				When:     time.Unix(1700000000, 0).UTC(),
			},
		}))
		require.Equal(t, "tag", runGit(t, repo, "cat-file", "-t", "refs/tags/annotated"))

		tag, err := g.GetAnnotatedTag(t.Context(), repo, "refs/tags/annotated")
		require.NoError(t, err)
		require.Equal(t, "annotated", tag.Name)
		require.Equal(t, head.String(), tag.TargetSHA.String())
		require.Equal(t, GitObjectTypeCommit, tag.TargetType)
		require.Equal(t, "release one", tag.Title)
		require.Equal(t, "Tagger", tag.Tagger.Identity.Name)

		tags, err := g.GetAnnotatedTags(t.Context(), repo, []sha.SHA{tag.Sha})
		require.NoError(t, err)
		require.Len(t, tags, 1)
		require.Equal(t, "annotated", tags[0].Name)
	})

	t.Run("duplicate fails", func(t *testing.T) {
		require.Error(t, g.CreateTag(t.Context(), repo, "light", head, nil))
	})

	t.Run("count", func(t *testing.T) {
		n, err := g.GetTagCount(t.Context(), repo)
		require.NoError(t, err)
		require.Equal(t, 2, n)

		_, err = g.GetTagCount(t.Context(), "")
		require.True(t, errors.Is(err, ErrRepositoryPathEmpty))
	})
}

func TestHashObjectAndGetBlob(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)

	content := "hello blob\n"
	blobSHA, err := g.HashObject(t.Context(), repo, strings.NewReader(content))
	require.NoError(t, err)
	require.Equal(t, content, runGit(t, repo, "cat-file", "-p", blobSHA.String())+"\n")

	t.Run("full content", func(t *testing.T) {
		r, err := GetBlob(t.Context(), repo, nil, blobSHA, 0)
		require.NoError(t, err)
		defer r.Content.Close()
		require.Equal(t, int64(len(content)), r.Size)
		require.Equal(t, int64(len(content)), r.ContentSize)
		data, err := io.ReadAll(r.Content)
		require.NoError(t, err)
		require.Equal(t, content, string(data))
	})

	t.Run("size limit", func(t *testing.T) {
		r, err := GetBlob(t.Context(), repo, nil, blobSHA, 5)
		require.NoError(t, err)
		defer r.Content.Close()
		require.Equal(t, int64(len(content)), r.Size)
		require.Equal(t, int64(5), r.ContentSize)
		data, err := io.ReadAll(r.Content)
		require.NoError(t, err)
		require.Equal(t, "hello", string(data))
	})

	t.Run("not a blob", func(t *testing.T) {
		treeSHA := sha.Must(runGit(t, repo, "rev-parse", "HEAD^{tree}"))
		_, err := GetBlob(t.Context(), repo, nil, treeSHA, 0)
		require.Error(t, err)
	})

	t.Run("unknown object", func(t *testing.T) {
		_, err := GetBlob(t.Context(), repo, nil, sha.Must("1111111111111111111111111111111111111111"), 0)
		require.Error(t, err)
	})
}

func TestConfig(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)

	require.True(t, errors.Is(g.Config(t.Context(), "", "a.b", "c"), ErrRepositoryPathEmpty))
	require.Error(t, g.Config(t.Context(), repo, "", "c"))

	require.NoError(t, g.Config(t.Context(), repo, "gitness.test", "value"))
	require.Equal(t, "value", runGit(t, repo, "config", "--local", "gitness.test"))

	// invalid key has no section and is rejected by git
	require.Error(t, g.Config(t.Context(), repo, "nosection", "value"))
}

func TestSanitizeCredentialURLs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no url", "just text", "just text"},
		{"no credentials", "see https://example.com/a/b", "see https://example.com/a/b"},
		{"user and password", "https://user:pass@example.com/x", "https://sanitized-credential@example.com/x"},
		{"user only", "clone https://token@example.com/x now", "clone https://sanitized-credential@example.com/x now"},
		{
			"two urls",
			"a https://u:p@h1.com b https://h2.com c https://u2@h3.com",
			"a https://sanitized-credential@h1.com b https://h2.com c https://sanitized-credential@h3.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, SanitizeCredentialURLs(tt.in))
		})
	}
}
