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
	"time"

	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

// historyRepo builds a repository with: first (a.txt), add b.txt, modify b.txt, rename b.txt -> c.txt.
func historyRepo(t *testing.T) (repo string, shas []string) {
	t.Helper()
	requireGit(t)

	repo = initRepoWithCommit(t)
	shas = append(shas, runGit(t, repo, "rev-parse", "HEAD"))
	shas = append(shas, commitFile(t, repo, "b.txt", "line1\n", "add b"))
	shas = append(shas, commitFile(t, repo, "b.txt", "line1\nline2\n", "modify b"))
	runGit(t, repo, "mv", "b.txt", "c.txt")
	runGit(t, repo, "commit", "-m", "rename b to c")
	shas = append(shas, runGit(t, repo, "rev-parse", "HEAD"))
	return repo, shas
}

func TestListCommits(t *testing.T) {
	g := &Git{}
	repo, shas := historyRepo(t)

	_, _, err := g.ListCommits(t.Context(), "", "main", 1, 10, false, CommitFilter{})
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	t.Run("all commits newest first", func(t *testing.T) {
		commits, renames, err := g.ListCommits(t.Context(), repo, "main", 0, 0, false, CommitFilter{})
		require.NoError(t, err)
		require.Nil(t, renames)
		require.Len(t, commits, 4)
		require.Equal(t, shas[3], commits[0].SHA.String())
		require.Equal(t, "rename b to c", commits[0].Title)
		require.Equal(t, shas[0], commits[3].SHA.String())
		require.Empty(t, commits[0].FileStats)
	})

	t.Run("pagination", func(t *testing.T) {
		page1, _, err := g.ListCommits(t.Context(), repo, "main", 1, 2, false, CommitFilter{})
		require.NoError(t, err)
		require.Len(t, page1, 2)
		page2, _, err := g.ListCommits(t.Context(), repo, "main", 2, 2, false, CommitFilter{})
		require.NoError(t, err)
		require.Len(t, page2, 2)
		require.Equal(t, shas[1], page2[0].SHA.String())
	})

	t.Run("with stats", func(t *testing.T) {
		commits, _, err := g.ListCommits(t.Context(), repo, "main", 1, 3, true, CommitFilter{})
		require.NoError(t, err)
		require.Len(t, commits, 3)

		renameStats := commits[0].FileStats
		require.Len(t, renameStats, 1)
		require.Equal(t, enum.FileDiffStatusRenamed, renameStats[0].ChangeType)
		require.Equal(t, "c.txt", renameStats[0].Path)
		require.Equal(t, "b.txt", renameStats[0].OldPath)

		modifyStats := commits[1].FileStats
		require.Len(t, modifyStats, 1)
		require.Equal(t, enum.FileDiffStatusModified, modifyStats[0].ChangeType)
		require.Equal(t, int64(1), modifyStats[0].Insertions)
		require.Equal(t, int64(0), modifyStats[0].Deletions)

		addStats := commits[2].FileStats
		require.Len(t, addStats, 1)
		require.Equal(t, enum.FileDiffStatusAdded, addStats[0].ChangeType)
		require.Equal(t, "b.txt", addStats[0].Path)
	})

	t.Run("path filter follows renames", func(t *testing.T) {
		commits, renames, err := g.ListCommits(t.Context(), repo, "main", 1, 10, false, CommitFilter{Path: "c.txt"})
		require.NoError(t, err)
		require.Len(t, commits, 1)
		require.Equal(t, shas[3], commits[0].SHA.String())
		require.Len(t, renames, 1)
		require.Equal(t, "b.txt", renames[0].OldPath)
		require.Equal(t, "c.txt", renames[0].Path)
		require.Equal(t, shas[3], renames[0].CommitSHABefore.String())
	})

	t.Run("path filter for old path drops rename commit", func(t *testing.T) {
		commits, renames, err := g.ListCommits(t.Context(), repo, "main", 1, 10, false, CommitFilter{Path: "b.txt"})
		require.NoError(t, err)
		require.NotEmpty(t, renames)
		for _, c := range commits {
			require.NotEqual(t, shas[3], c.SHA.String())
		}
	})

	t.Run("filters", func(t *testing.T) {
		commits, _, err := g.ListCommits(t.Context(), repo, "main", 1, 10, false, CommitFilter{AfterRef: shas[1]})
		require.NoError(t, err)
		require.Len(t, commits, 2)

		commits, _, err = g.ListCommits(t.Context(), repo, "main", 1, 10, false, CommitFilter{Author: "test"})
		require.NoError(t, err)
		require.Len(t, commits, 4)

		commits, _, err = g.ListCommits(t.Context(), repo, "main", 1, 10, false,
			CommitFilter{Committer: "te.t", Regex: true})
		require.NoError(t, err)
		require.Len(t, commits, 4)

		commits, _, err = g.ListCommits(t.Context(), repo, "main", 1, 10, false, CommitFilter{Author: "nobody"})
		require.NoError(t, err)
		require.Empty(t, commits)

		since := time.Now().Add(time.Hour).Unix()
		commits, _, err = g.ListCommits(t.Context(), repo, "main", 1, 10, false, CommitFilter{Since: since})
		require.NoError(t, err)
		require.Empty(t, commits)

		until := time.Now().Add(time.Hour).Unix()
		commits, _, err = g.ListCommits(t.Context(), repo, "main", 1, 10, false, CommitFilter{Until: until})
		require.NoError(t, err)
		require.Len(t, commits, 4)
	})

	t.Run("unknown ref yields empty list", func(t *testing.T) {
		commits, _, err := g.ListCommits(t.Context(), repo, "does-not-exist", 1, 10, false, CommitFilter{})
		require.NoError(t, err)
		require.Empty(t, commits)
	})
}

func TestListCommitSHAs(t *testing.T) {
	g := &Git{}
	repo, shas := historyRepo(t)

	got, err := g.ListCommitSHAs(t.Context(), repo, nil, "main", 1, 2, CommitFilter{})
	require.NoError(t, err)
	require.Equal(t, []string{shas[3], shas[2]}, shaStrings(got))
}

func shaStrings(in []sha.SHA) []string {
	out := make([]string, len(in))
	for i := range in {
		out[i] = in[i].String()
	}
	return out
}

func TestGetCommitHelpers(t *testing.T) {
	g := &Git{}
	repo, shas := historyRepo(t)

	t.Run("GetCommitFromRev", func(t *testing.T) {
		_, err := g.GetCommitFromRev(t.Context(), "", "main")
		require.ErrorIs(t, err, ErrRepositoryPathEmpty)

		c, err := g.GetCommitFromRev(t.Context(), repo, "main")
		require.NoError(t, err)
		require.Equal(t, shas[3], c.SHA.String())
		require.Equal(t, "rename b to c", c.Title)
		require.Len(t, c.ParentSHAs, 1)
		require.Equal(t, shas[2], c.ParentSHAs[0].String())
		require.Equal(t, "test", c.Author.Identity.Name)

		_, err = g.GetCommitFromRev(t.Context(), repo, "missing-rev")
		require.Error(t, err)
	})

	t.Run("GetCommits", func(t *testing.T) {
		_, err := g.GetCommits(t.Context(), "", nil)
		require.ErrorIs(t, err, ErrRepositoryPathEmpty)

		commits, err := g.GetCommits(t.Context(), repo, []sha.SHA{sha.Must(shas[0]), sha.Must(shas[1])})
		require.NoError(t, err)
		require.Len(t, commits, 2)
		require.Equal(t, shas[0], commits[0].SHA.String())
		require.Equal(t, shas[1], commits[1].SHA.String())
	})

	t.Run("GetLatestCommit", func(t *testing.T) {
		c, err := GetLatestCommit(t.Context(), repo, "main", "a.txt")
		require.NoError(t, err)
		require.Equal(t, shas[0], c.SHA.String())

		c, err = GetLatestCommit(t.Context(), repo, "main", "c.txt")
		require.NoError(t, err)
		require.Equal(t, shas[3], c.SHA.String())

		_, err = GetLatestCommit(t.Context(), repo, "main", "nope.txt")
		require.Error(t, err)

		_, err = GetLatestCommit(t.Context(), repo, "missing-rev", "a.txt")
		require.Error(t, err)
	})
}

func TestGetCommitDivergences_Counts(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	runGit(t, repo, "checkout", "-q", "-b", "feature")
	commitFile(t, repo, "f1.txt", "1\n", "f1")
	commitFile(t, repo, "f2.txt", "2\n", "f2")
	runGit(t, repo, "checkout", "-q", "main")
	commitFile(t, repo, "m1.txt", "1\n", "m1")

	_, err := g.GetCommitDivergences(t.Context(), "", nil, 0)
	require.ErrorIs(t, err, ErrRepositoryPathEmpty)

	got, err := g.GetCommitDivergences(t.Context(), repo, []CommitDivergenceRequest{
		{From: "feature", To: "main"},
		{From: "main", To: "feature"},
		{From: "main", To: "main"},
		{From: "main", To: "missing"},
	}, 0)
	require.NoError(t, err)
	require.Equal(t, []CommitDivergence{
		{Ahead: 2, Behind: 1},
		{Ahead: 1, Behind: 2},
		{Ahead: 0, Behind: 0},
		{Ahead: -1, Behind: -1},
	}, got)

	limited, err := g.GetCommitDivergences(t.Context(), repo,
		[]CommitDivergenceRequest{{From: "feature", To: "main"}}, 1)
	require.NoError(t, err)
	require.Equal(t, int32(1), limited[0].Ahead+limited[0].Behind)
}

func TestConvertFileDiffStatus(t *testing.T) {
	tests := map[string]enum.FileDiffStatus{
		"A\tfile":      enum.FileDiffStatusAdded,
		"C100\ta\tb":   enum.FileDiffStatusCopied,
		"D\tfile":      enum.FileDiffStatusDeleted,
		"M\tfile":      enum.FileDiffStatusModified,
		"R100\ta\tb":   enum.FileDiffStatusRenamed,
		"X\tunknown":   enum.FileDiffStatusUndefined,
		"":             enum.FileDiffStatusUndefined,
		"Modified":     enum.FileDiffStatusModified,
		"zzz\tnothing": enum.FileDiffStatusUndefined,
	}
	for in, want := range tests {
		t.Run(strings.ReplaceAll(in, "\t", " "), func(t *testing.T) {
			require.Equal(t, want, convertFileDiffStatus(t.Context(), in))
		})
	}
}

func TestParseLinesToSlice(t *testing.T) {
	require.Nil(t, parseLinesToSlice(nil))
	require.Equal(t, []string{"a", "b"}, parseLinesToSlice([]byte("a\nb\n")))
	require.Equal(t, []string{"a"}, parseLinesToSlice([]byte("\n a\n")[1:]))
}

func TestCleanupCommitsForRename(t *testing.T) {
	c1 := Commit{SHA: sha.Must("1111111111111111111111111111111111111111")}
	c2 := Commit{SHA: sha.Must("2222222222222222222222222222222222222222")}

	require.Empty(t, cleanupCommitsForRename(nil, nil, "p"))

	commits := []Commit{c1, c2}
	require.Equal(t, commits, cleanupCommitsForRename(commits, nil, "p"))

	renames := []PathRenameDetails{{OldPath: "old", Path: "new", CommitSHABefore: c1.SHA}}
	require.Equal(t, []Commit{c2}, cleanupCommitsForRename(commits, renames, "old"))
	require.Equal(t, commits, cleanupCommitsForRename(commits, renames, "new"))
}

func TestGetRenameDetails(t *testing.T) {
	repo, shas := historyRepo(t)

	commits, err := CatFileCommits(t.Context(), repo, nil, []sha.SHA{sha.Must(shas[3])})
	require.NoError(t, err)

	got, err := getRenameDetails(t.Context(), repo, nil, "c.txt")
	require.NoError(t, err)
	require.Empty(t, got)

	got, err = getRenameDetails(t.Context(), repo, commits, "c.txt")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "b.txt", got[0].OldPath)

	// last commit of a multi commit list is checked for renames too
	all, err := CatFileCommits(t.Context(), repo, nil, []sha.SHA{sha.Must(shas[3]), sha.Must(shas[0])})
	require.NoError(t, err)
	got, err = getRenameDetails(t.Context(), repo, all, "c.txt")
	require.NoError(t, err)
	require.Len(t, got, 1)

	got, err = getRenameDetails(t.Context(), repo, all, "unrelated.txt")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestChangeInfoParsing_Errors(t *testing.T) {
	repo, shas := historyRepo(t)
	missing := sha.Must("1111111111111111111111111111111111111111")

	_, err := getChangeInfoTypes(t.Context(), repo, missing)
	require.Error(t, err)
	_, err = getChangeInfoChanges(t.Context(), repo, missing)
	require.Error(t, err)
	_, err = getCommitFileStats(t.Context(), repo, missing)
	require.Error(t, err)

	// merge commits and the initial commit
	stats, err := getCommitFileStats(t.Context(), repo, sha.Must(shas[0]))
	require.NoError(t, err)
	require.Len(t, stats, 1)
}

func TestChangeInfoChanges_BinaryAndRename(t *testing.T) {
	requireGit(t)
	repo := initRepoWithCommit(t)
	writeRepoFile(t, repo, "bin.dat", "\x00\x01\x02binary\x00")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "binary")
	head := sha.Must(runGit(t, repo, "rev-parse", "HEAD"))

	changes, err := getChangeInfoChanges(t.Context(), repo, head)
	require.NoError(t, err)
	require.Equal(t, map[string]changeInfoChange{"bin.dat": {}}, changes)

	runGit(t, repo, "mv", "bin.dat", "moved.dat")
	runGit(t, repo, "commit", "-m", "move")
	head = sha.Must(runGit(t, repo, "rev-parse", "HEAD"))
	changes, err = getChangeInfoChanges(t.Context(), repo, head)
	require.NoError(t, err)
	require.Contains(t, changes, "moved.dat")
}
