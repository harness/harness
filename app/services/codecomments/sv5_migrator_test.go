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

package codecomments

import (
	"context"
	"errors"
	"testing"

	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type sv5Fetcher struct {
	output map[string]git.GetDiffHunkHeadersOutput
	err    map[string]error
	calls  []git.GetDiffHunkHeadersParams
}

func (f *sv5Fetcher) GetDiffHunkHeaders(
	_ context.Context, p git.GetDiffHunkHeadersParams,
) (git.GetDiffHunkHeadersOutput, error) {
	f.calls = append(f.calls, p)
	if err := f.err[p.SourceCommitSHA]; err != nil {
		return git.GetDiffHunkHeadersOutput{}, err
	}
	return f.output[p.SourceCommitSHA], nil
}

func sv5Comment(id int64, path string, fields func(*types.CodeComment)) *types.CodeComment {
	cc := &types.CodeComment{ID: id}
	cc.Path = path
	cc.MergeBaseSHA = "base-old"
	cc.SourceSHA = "src-old"
	cc.LineNew, cc.SpanNew = 10, 2
	cc.LineOld, cc.SpanOld = 10, 2
	if fields != nil {
		fields(cc)
	}
	return cc
}

func sv5File(oldName, newName string, ext map[string]string, hunks ...git.HunkHeader) git.DiffFileHunkHeaders {
	return git.DiffFileHunkHeaders{
		FileHeader:  git.DiffFileHeader{OldName: oldName, NewName: newName, Extensions: ext},
		HunkHeaders: hunks,
	}
}

func TestSv5MigrateOldMovesLinesAndUpdatesBaseSHA(t *testing.T) {
	fetcher := &sv5Fetcher{output: map[string]git.GetDiffHunkHeadersOutput{
		"base-old": {Files: []git.DiffFileHunkHeaders{
			// Three lines were inserted before the comment.
			sv5File("a.go", "a.go", nil, git.HunkHeader{OldLine: 2, OldSpan: 0, NewLine: 2, NewSpan: 3}),
		}},
	}}
	migrator := &Migrator{hunkHeaderFetcher: fetcher}

	cc := sv5Comment(1, "a.go", nil)
	migrator.MigrateOld(context.Background(), "repo-uid", "base-new", []*types.CodeComment{cc})

	require.Equal(t, "base-new", cc.MergeBaseSHA)
	require.Equal(t, "src-old", cc.SourceSHA, "the source side is untouched")
	require.Equal(t, 13, cc.LineOld)
	require.False(t, cc.Outdated)
	require.Equal(t, "repo-uid", fetcher.calls[0].RepoUID)
	require.Equal(t, "base-old", fetcher.calls[0].SourceCommitSHA)
	require.Equal(t, "base-new", fetcher.calls[0].TargetCommitSHA)
}

func TestSv5MigrateNoComments(t *testing.T) {
	fetcher := &sv5Fetcher{}
	migrator := &Migrator{hunkHeaderFetcher: fetcher}
	migrator.MigrateNew(context.Background(), "repo-uid", "new", nil)
	migrator.MigrateOld(context.Background(), "repo-uid", "new", []*types.CodeComment{})
	require.Empty(t, fetcher.calls)
}

func TestSv5MigrateFileLevelChanges(t *testing.T) {
	fetcher := &sv5Fetcher{output: map[string]git.GetDiffHunkHeadersOutput{
		"src-old": {Files: []git.DiffFileHunkHeaders{
			sv5File("renamed-old.go", "renamed-new.go", nil),
			sv5File("deleted.go", "deleted.go", map[string]string{gitenum.DiffExtHeaderDeletedFileMode: "100644"}),
			sv5File("added.go", "added.go", map[string]string{gitenum.DiffExtHeaderNewFileMode: "100644"}),
			// Only the new name is known to the comments: found through the new name.
			sv5File("moved-away.go", "moved-here.go", nil),
		}},
	}}
	migrator := &Migrator{hunkHeaderFetcher: fetcher}

	renamed := sv5Comment(1, "renamed-old.go", nil)
	deleted := sv5Comment(2, "deleted.go", nil)
	added := sv5Comment(3, "added.go", nil)
	moved := sv5Comment(4, "moved-here.go", nil)
	untouched := sv5Comment(5, "untouched.go", nil)

	migrator.MigrateNew(context.Background(), "repo-uid", "src-new",
		[]*types.CodeComment{renamed, deleted, added, moved, untouched})

	require.Equal(t, "renamed-new.go", renamed.Path, "comment follows the file rename")
	require.False(t, renamed.Outdated)
	require.Equal(t, "src-new", renamed.SourceSHA)

	require.True(t, deleted.Outdated, "comments of deleted files are outdated")
	require.Equal(t, "src-old", deleted.SourceSHA)
	require.True(t, added.Outdated, "comments of newly added files are outdated")
	require.Equal(t, "src-old", added.SourceSHA)

	require.False(t, moved.Outdated)
	require.Equal(t, "moved-here.go", moved.Path)
	require.Equal(t, "src-new", moved.SourceSHA)

	require.Equal(t, "src-new", untouched.SourceSHA, "files not in the diff are unchanged but on the new SHA")
	require.Equal(t, 10, untouched.LineNew)
}

func TestSv5MigrateOverlappingHunkOutdatesAndRestoresComment(t *testing.T) {
	fetcher := &sv5Fetcher{output: map[string]git.GetDiffHunkHeadersOutput{
		"src-old": {Files: []git.DiffFileHunkHeaders{
			// The hunk rewrites the commented lines, preceded by a hunk that would move them.
			sv5File("a.go", "a.go", nil,
				git.HunkHeader{OldLine: 1, OldSpan: 0, NewLine: 1, NewSpan: 2},
				git.HunkHeader{OldLine: 10, OldSpan: 2, NewLine: 10, NewSpan: 1},
			),
		}},
	}}
	migrator := &Migrator{hunkHeaderFetcher: fetcher}

	cc := sv5Comment(1, "a.go", nil)
	migrator.MigrateNew(context.Background(), "repo-uid", "src-new", []*types.CodeComment{cc})

	require.True(t, cc.Outdated)
	require.Equal(t, 10, cc.LineNew, "the original values are restored")
	require.Equal(t, 2, cc.SpanNew)
	require.Equal(t, "src-old", cc.SourceSHA)
}

func TestSv5MigrateDiffFailures(t *testing.T) {
	fetcher := &sv5Fetcher{
		err: map[string]error{
			"src-gone":  gitnesserrors.NotFound("commit"),
			"src-error": errors.New("boom"),
		},
		output: map[string]git.GetDiffHunkHeadersOutput{"src-ok": {}},
	}
	migrator := &Migrator{hunkHeaderFetcher: fetcher}

	gone := sv5Comment(1, "a.go", func(c *types.CodeComment) { c.SourceSHA = "src-gone" })
	failed := sv5Comment(2, "a.go", func(c *types.CodeComment) { c.SourceSHA = "src-error" })
	ok := sv5Comment(3, "a.go", func(c *types.CodeComment) { c.SourceSHA = "src-ok" })

	migrator.MigrateNew(context.Background(), "repo-uid", "src-new", []*types.CodeComment{gone, failed, ok})

	require.True(t, gone.Outdated, "a commit that doesn't exist anymore outdates its comments")
	require.False(t, failed.Outdated, "other failures leave the comment as it is")
	require.Equal(t, "src-error", failed.SourceSHA)
	require.Equal(t, "src-new", ok.SourceSHA)
}
