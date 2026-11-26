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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchFiles(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	writeRepoFile(t, repo, "docs/one.md", "one\n")
	writeRepoFile(t, repo, "docs/two.md", "two\n")
	writeRepoFile(t, repo, "docs/empty.md", "")
	writeRepoFile(t, repo, "docs/big.md", strings.Repeat("x", 100)+"\n")
	writeRepoFile(t, repo, "docs/other.txt", "other\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "docs")

	t.Run("matches pattern", func(t *testing.T) {
		files, err := g.MatchFiles(t.Context(), repo, "main", "docs", "*.md", 50)
		require.NoError(t, err)
		got := map[string]string{}
		for _, f := range files {
			got[f.Path] = string(f.Content)
		}
		require.Equal(t, map[string]string{"docs/one.md": "one\n", "docs/two.md": "two\n"}, got)
	})

	t.Run("bad pattern", func(t *testing.T) {
		_, err := g.MatchFiles(t.Context(), repo, "main", "docs", "[", 50)
		require.Error(t, err)
	})

	t.Run("bad path", func(t *testing.T) {
		_, err := g.MatchFiles(t.Context(), repo, "main", "missing-dir", "*", 50)
		require.Error(t, err)
	})
}

func TestGetSubModules(t *testing.T) {
	content := `[submodule "lib"]
	path = vendor/lib
	url = https://example.com/lib.git
[submodule "other"]
	path = vendor/other
	url = https://example.com/other.git
`
	mods, err := GetSubModules(&BlobReader{Content: nopReadCloser{strings.NewReader(content)}})
	require.NoError(t, err)
	require.Equal(t, map[string]*Submodule{
		"vendor/lib":   {Name: "vendor/lib", URL: "https://example.com/lib.git"},
		"vendor/other": {Name: "vendor/other", URL: "https://example.com/other.git"},
	}, mods)
}

func TestGetSubmodule(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	commitFile(t, repo, ".gitmodules", "[submodule \"lib\"]\n\tpath = vendor/lib\n\turl = https://example.com/lib.git\n",
		"add gitmodules")

	_, err := g.GetSubmodule(t.Context(), "", "main", "vendor/lib")
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	sm, err := g.GetSubmodule(t.Context(), repo, "main", "vendor/lib")
	require.NoError(t, err)
	require.Equal(t, &Submodule{Name: "vendor/lib", URL: "https://example.com/lib.git"}, sm)

	sm, err = g.GetSubmodule(t.Context(), repo, "main", "vendor/unknown")
	require.NoError(t, err)
	require.Nil(t, sm)

	_, err = g.GetSubmodule(t.Context(), repo, "no-such-ref", "vendor/lib")
	require.Error(t, err)
}

type nopReadCloser struct{ *strings.Reader }

func (nopReadCloser) Close() error { return nil }
