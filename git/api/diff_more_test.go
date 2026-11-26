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
	"strings"
	"testing"

	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

// diffRepo builds main (a.txt, del.txt, old.txt) and feature (modified a.txt, added new.txt,
// deleted del.txt, renamed old.txt -> renamed.txt).
func diffRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)

	repo := initRepoWithCommit(t)
	writeRepoFile(t, repo, "a.txt", "one\ntwo\nthree\nfour\nfive\n")
	writeRepoFile(t, repo, "del.txt", "gone\n")
	writeRepoFile(t, repo, "old.txt", "rename me\nsecond line\nthird line\nfourth line\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "base")

	runGit(t, repo, "checkout", "-q", "-b", "feature")
	writeRepoFile(t, repo, "a.txt", "one\nTWO\nthree\nfour\nfive\n")
	writeRepoFile(t, repo, "new.txt", "new1\nnew2\n")
	runGit(t, repo, "rm", "-q", "del.txt")
	runGit(t, repo, "mv", "old.txt", "renamed.txt")
	writeRepoFile(t, repo, "renamed.txt", "rename me\nSECOND line\nthird line\nfourth line\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "feature changes")
	runGit(t, repo, "checkout", "-q", "main")
	return repo
}

func TestRawDiff(t *testing.T) {
	g := &Git{}
	repo := diffRepo(t)

	raw := func(mergeBase, ignoreWS bool, files ...FileDiffRequest) (string, error) {
		var buf bytes.Buffer
		err := g.RawDiff(t.Context(), &buf, repo, "main", "feature", mergeBase, ignoreWS, nil, files...)
		return buf.String(), err
	}

	require.ErrorIs(t, g.RawDiff(t.Context(), &bytes.Buffer{}, "", "a", "b", false, false, nil), ErrRepositoryPathEmpty)

	t.Run("full diff", func(t *testing.T) {
		out, err := raw(false, false)
		require.NoError(t, err)
		require.Contains(t, out, "diff --git a/a.txt b/a.txt")
		require.Contains(t, out, "-two")
		require.Contains(t, out, "+TWO")
		require.Contains(t, out, "new file mode")
		require.Contains(t, out, "deleted file mode")
		require.Contains(t, out, "rename from old.txt")
	})

	t.Run("merge base and ignore whitespace", func(t *testing.T) {
		out, err := raw(true, true)
		require.NoError(t, err)
		require.Contains(t, out, "+TWO")
	})

	t.Run("path filter", func(t *testing.T) {
		out, err := raw(false, false, FileDiffRequest{Path: "new.txt"})
		require.NoError(t, err)
		require.Contains(t, out, "new.txt")
		require.NotContains(t, out, "a.txt")
	})

	t.Run("per file line ranges", func(t *testing.T) {
		out, err := raw(false, false,
			FileDiffRequest{Path: "a.txt", StartLine: 1, EndLine: 3},
			FileDiffRequest{Path: "new.txt"},
		)
		require.NoError(t, err)
		require.Contains(t, out, "a.txt")
		require.Contains(t, out, "new.txt")
	})

	t.Run("unknown ref", func(t *testing.T) {
		var buf bytes.Buffer
		err := g.RawDiff(t.Context(), &buf, repo, "main", "no-such-ref", false, false, nil)
		require.Error(t, err)

		err = g.RawDiff(t.Context(), &buf, repo, "main", "no-such-ref", true, false, nil)
		require.Error(t, err)
	})

	t.Run("annotated tag refs", func(t *testing.T) {
		runGit(t, repo, "tag", "-a", "-m", "tag message", "v1", "feature")
		var buf bytes.Buffer
		require.NoError(t, g.RawDiff(t.Context(), &buf, repo, "main", "v1", false, false, nil))
		require.Contains(t, buf.String(), "+TWO")
	})
}

func TestCommitDiff(t *testing.T) {
	g := &Git{}
	repo := diffRepo(t)

	require.ErrorIs(t, g.CommitDiff(t.Context(), "", "main", false, &bytes.Buffer{}), ErrRepositoryPathEmpty)
	require.Error(t, g.CommitDiff(t.Context(), repo, "", false, &bytes.Buffer{}))

	var buf bytes.Buffer
	require.NoError(t, g.CommitDiff(t.Context(), repo, "feature", false, &buf))
	require.Contains(t, buf.String(), "+TWO")

	buf.Reset()
	require.NoError(t, g.CommitDiff(t.Context(), repo, "feature", true, &buf))
	require.Contains(t, buf.String(), "+TWO")

	require.Error(t, g.CommitDiff(t.Context(), repo, "no-such-rev", false, &buf))
}

func TestDiffShortStatAndFileName(t *testing.T) {
	g := &Git{}
	repo := diffRepo(t)

	_, err := g.DiffShortStat(t.Context(), "", "main", "feature", false, false)
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	stat, err := g.DiffShortStat(t.Context(), repo, "main", "feature", false, false)
	require.NoError(t, err)
	require.Equal(t, 4, stat.Files)
	require.Equal(t, 4, stat.Additions)
	require.Equal(t, 3, stat.Deletions)

	merged, err := g.DiffShortStat(t.Context(), repo, "main", "feature", true, true)
	require.NoError(t, err)
	require.Equal(t, 4, merged.Files)

	fromEmpty, err := g.DiffShortStat(t.Context(), repo, types.NilSHA, "main", false, false)
	require.NoError(t, err)
	require.Positive(t, fromEmpty.Files)
	fromBlank, err := g.DiffShortStat(t.Context(), repo, "", "main", false, false)
	require.NoError(t, err)
	require.Equal(t, fromEmpty, fromBlank)

	same, err := g.DiffShortStat(t.Context(), repo, "main", "main", false, false)
	require.NoError(t, err)
	require.Equal(t, DiffShortStat{}, same)

	_, err = g.DiffShortStat(t.Context(), repo, "main", "no-such-ref", false, false)
	require.Error(t, err)

	names, err := g.DiffFileName(t.Context(), repo, "main", "feature", false, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"a.txt", "del.txt", "new.txt", "renamed.txt"}, names)

	names, err = g.DiffFileName(t.Context(), repo, "main", "feature", true, true)
	require.NoError(t, err)
	require.Contains(t, names, "a.txt")

	_, err = g.DiffFileName(t.Context(), repo, "main", "no-such-ref", false, false)
	require.Error(t, err)
}

func TestGetDiffHunkHeaders(t *testing.T) {
	g := &Git{}
	repo := diffRepo(t)

	_, err := g.GetDiffHunkHeaders(t.Context(), "", "main", "feature")
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	// the base is the source and the head the target
	headers, err := g.GetDiffHunkHeaders(t.Context(), repo, "feature", "main")
	require.NoError(t, err)
	require.NotEmpty(t, headers)
	var paths []string
	for _, h := range headers {
		paths = append(paths, h.FileHeader.NewFileName)
	}
	require.Contains(t, paths, "a.txt")

	_, err = g.GetDiffHunkHeaders(t.Context(), repo, "feature", "no-such-ref")
	require.Error(t, err)
}

func TestDiffCut(t *testing.T) {
	g := &Git{}
	repo := diffRepo(t)

	_, _, err := g.DiffCut(t.Context(), "", "main", "feature", "a.txt", false, parser.DiffCutParams{})
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	t.Run("modified file new side", func(t *testing.T) {
		header, hunk, err := g.DiffCut(t.Context(), repo, "main", "feature", "a.txt", false, parser.DiffCutParams{
			LineStart: 2, LineStartNew: true, LineEnd: 2, LineEndNew: true, BeforeLines: 1, AfterLines: 1,
		})
		require.NoError(t, err)
		require.NotEmpty(t, hunk.Lines)
		require.NotZero(t, header.NewLine)
		require.Contains(t, strings.Join(hunk.Lines, "\n"), "+TWO")
	})

	t.Run("modified file old side", func(t *testing.T) {
		_, hunk, err := g.DiffCut(t.Context(), repo, "main", "feature", "a.txt", true, parser.DiffCutParams{
			LineStart: 2, LineEnd: 2,
		})
		require.NoError(t, err)
		require.Contains(t, strings.Join(hunk.Lines, "\n"), "-two")
	})

	t.Run("added file", func(t *testing.T) {
		_, hunk, err := g.DiffCut(t.Context(), repo, "main", "feature", "new.txt", false, parser.DiffCutParams{
			LineStart: 1, LineStartNew: true, LineEnd: 2, LineEndNew: true,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"+new1", "+new2"}, hunk.Lines)
	})

	t.Run("deleted file", func(t *testing.T) {
		_, hunk, err := g.DiffCut(t.Context(), repo, "main", "feature", "del.txt", false, parser.DiffCutParams{
			LineStart: 1, LineEnd: 1,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"-gone"}, hunk.Lines)
	})

	t.Run("renamed file", func(t *testing.T) {
		_, hunk, err := g.DiffCut(t.Context(), repo, "main", "feature", "renamed.txt", false, parser.DiffCutParams{
			LineStart: 2, LineStartNew: true, LineEnd: 2, LineEndNew: true,
		})
		require.NoError(t, err)
		require.NotEmpty(t, hunk.Lines)

		// wrong side for a rename
		_, _, err = g.DiffCut(t.Context(), repo, "main", "feature", "old.txt", false, parser.DiffCutParams{
			LineStart: 1, LineStartNew: true, LineEnd: 1, LineEndNew: true,
		})
		require.Error(t, err)
		_, _, err = g.DiffCut(t.Context(), repo, "main", "feature", "renamed.txt", false, parser.DiffCutParams{
			LineStart: 1, LineEnd: 1,
		})
		require.Error(t, err)
	})

	t.Run("unknown path", func(t *testing.T) {
		_, _, err := g.DiffCut(t.Context(), repo, "main", "feature", "nope.txt", false, parser.DiffCutParams{})
		require.Error(t, err)
	})

	t.Run("unknown ref", func(t *testing.T) {
		_, _, err := g.DiffCut(t.Context(), repo, "main", "no-such-ref", "a.txt", false, parser.DiffCutParams{})
		require.Error(t, err)
	})
}

func TestParseDiffStat(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    DiffShortStat
		wantErr bool
	}{
		{"empty", "", DiffShortStat{}, false},
		{"newline", "\n", DiffShortStat{}, false},
		{"all", " 3 files changed, 10 insertions(+), 4 deletions(-)\n", DiffShortStat{3, 10, 4}, false},
		{"single", " 1 file changed, 1 insertion(+), 1 deletion(-)\n", DiffShortStat{1, 1, 1}, false},
		{"insertions only", " 2 files changed, 5 insertions(+)\n", DiffShortStat{2, 5, 0}, false},
		{"deletions only", " 2 files changed, 6 deletions(-)\n", DiffShortStat{2, 0, 6}, false},
		{"garbage", "nothing useful", DiffShortStat{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDiffStat(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetDiffShortStat_Args(t *testing.T) {
	repo := diffRepo(t)

	stat, err := GetDiffShortStat(t.Context(), repo, false, sha.EmptyTree.String(), "feature")
	require.NoError(t, err)
	require.Positive(t, stat.Files)

	_, err = GetDiffShortStat(t.Context(), repo, true, "no-such-ref")
	require.Error(t, err)
}
