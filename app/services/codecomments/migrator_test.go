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

package codecomments

import (
	"context"
	"testing"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

// nolint:gocognit // it's a unit test
func TestMigrator(t *testing.T) {
	const (
		repoUID    = "not-important"
		fileName   = "blah" // file name is fixed across all the tests.
		shaSrcOld  = "old"
		shaSrcNew  = "new"
		shaBaseOld = "base-old"
		shaBaseNew = "base-new"
	)

	type position struct {
		lineOld, spanOld, lineNew, spanNew int
		mergeBaseSHA, sourceSHA            string
		outdated                           bool
	}

	tests := []struct {
		name      string
		headers   []git.HunkHeader
		rebase    bool
		positions []position
		expected  []position
	}{
		{
			name:    "source:no-hunks",
			headers: nil,
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
			},
		},
		{
			name: "source:lines-added-before-two-comments",
			headers: []git.HunkHeader{
				{OldLine: 0, OldSpan: 0, NewLine: 10, NewSpan: 10},
			},
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 40, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
				{lineOld: 50, spanOld: 1, lineNew: 60, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
			},
		},
		{
			name: "source:lines-added-between-two-comments",
			headers: []git.HunkHeader{
				{OldLine: 40, OldSpan: 0, NewLine: 40, NewSpan: 40},
			},
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
				{lineOld: 50, spanOld: 1, lineNew: 90, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
			},
		},
		{
			name: "source:lines-added-after-two-comments",
			headers: []git.HunkHeader{
				{OldLine: 60, OldSpan: 0, NewLine: 60, NewSpan: 200},
			},
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
			},
		},
		{
			name: "source:modified-second-comment",
			headers: []git.HunkHeader{
				{OldLine: 50, OldSpan: 1, NewLine: 50, NewSpan: 1},
			},
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
				{lineOld: 70, spanOld: 1, lineNew: 70, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcOld,
					outdated: true},
				{lineOld: 70, spanOld: 1, lineNew: 70, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
			},
		},
		{
			name: "source:modified-second-comment;also-removed-10-lines-at-1",
			headers: []git.HunkHeader{
				{OldLine: 1, OldSpan: 10, NewLine: 0, NewSpan: 0},
				{OldLine: 50, OldSpan: 1, NewLine: 40, NewSpan: 1},
			},
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
				{lineOld: 70, spanOld: 1, lineNew: 70, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 20, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcOld,
					outdated: true},
				{lineOld: 70, spanOld: 1, lineNew: 60, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
			},
		},
		{
			name: "source:modified-second-comment;also-added-10-lines-at-1",
			headers: []git.HunkHeader{
				{OldLine: 0, OldSpan: 0, NewLine: 1, NewSpan: 10},
				{OldLine: 50, OldSpan: 1, NewLine: 60, NewSpan: 1},
			},
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
				{lineOld: 70, spanOld: 1, lineNew: 70, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 40, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcOld,
					outdated: true},
				{lineOld: 70, spanOld: 1, lineNew: 80, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcNew},
			},
		},
		{
			name:    "merge-base:no-hunks",
			headers: nil,
			rebase:  true,
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
			},
		},
		{
			name: "merge-base:lines-added-before-two-comments",
			headers: []git.HunkHeader{
				{OldLine: 0, OldSpan: 0, NewLine: 10, NewSpan: 10},
			},
			rebase: true,
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
			},
			expected: []position{
				{lineOld: 40, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
				{lineOld: 60, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
			},
		},
		{
			name: "merge-base:lines-added-between-two-comments",
			headers: []git.HunkHeader{
				{OldLine: 40, OldSpan: 0, NewLine: 40, NewSpan: 40},
			},
			rebase: true,
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
				{lineOld: 90, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
			},
		},
		{
			name: "merge-base:lines-added-after-two-comments",
			headers: []git.HunkHeader{
				{OldLine: 60, OldSpan: 0, NewLine: 60, NewSpan: 200},
			},
			rebase: true,
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
			},
		},
		{
			name: "merge-base:modified-second-comment",
			headers: []git.HunkHeader{
				{OldLine: 50, OldSpan: 1, NewLine: 50, NewSpan: 1},
			},
			rebase: true,
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
				{lineOld: 70, spanOld: 1, lineNew: 70, spanNew: 1},
			},
			expected: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcOld,
					outdated: true},
				{lineOld: 70, spanOld: 1, lineNew: 70, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
			},
		},
		{
			name: "merge-base:modified-second-comment;also-removed-10-lines-at-1",
			headers: []git.HunkHeader{
				{OldLine: 1, OldSpan: 10, NewLine: 0, NewSpan: 0},
				{OldLine: 50, OldSpan: 1, NewLine: 40, NewSpan: 1},
			},
			rebase: true,
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
				{lineOld: 70, spanOld: 1, lineNew: 70, spanNew: 1},
			},
			expected: []position{
				{lineOld: 20, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcOld,
					outdated: true},
				{lineOld: 60, spanOld: 1, lineNew: 70, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
			},
		},
		{
			name: "merge-base:modified-second-comment;also-added-10-lines-at-1",
			headers: []git.HunkHeader{
				{OldLine: 0, OldSpan: 0, NewLine: 1, NewSpan: 10},
				{OldLine: 50, OldSpan: 1, NewLine: 60, NewSpan: 1},
			},
			rebase: true,
			positions: []position{
				{lineOld: 30, spanOld: 1, lineNew: 30, spanNew: 1},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1},
				{lineOld: 70, spanOld: 1, lineNew: 70, spanNew: 1},
			},
			expected: []position{
				{lineOld: 40, spanOld: 1, lineNew: 30, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
				{lineOld: 50, spanOld: 1, lineNew: 50, spanNew: 1, mergeBaseSHA: shaBaseOld, sourceSHA: shaSrcOld,
					outdated: true},
				{lineOld: 80, spanOld: 1, lineNew: 70, spanNew: 1, mergeBaseSHA: shaBaseNew, sourceSHA: shaSrcOld},
			},
		},
	}

	ctx := context.Background()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := testHunkHeaderFetcher{
				fileName: fileName,
				headers:  test.headers,
			}

			m := &Migrator{
				hunkHeaderFetcher: f,
			}

			comments := make([]*types.CodeComment, len(test.positions))
			for i, pos := range test.positions {
				comments[i] = &types.CodeComment{
					ID: int64(i),
					CodeCommentFields: types.CodeCommentFields{
						Outdated:     pos.outdated,
						MergeBaseSHA: shaBaseOld,
						SourceSHA:    shaSrcOld,
						Path:         fileName,
						LineNew:      pos.lineNew,
						SpanNew:      pos.spanNew,
						LineOld:      pos.lineOld,
						SpanOld:      pos.spanOld,
					},
				}
			}

			if test.rebase {
				m.MigrateOld(ctx, repoUID, shaBaseNew, comments)
			} else {
				m.MigrateNew(ctx, repoUID, shaSrcNew, comments)
			}

			for i, expPos := range test.expected {
				if expPos.outdated != comments[i].Outdated {
					t.Errorf("comment=%d, outdated mismatch", i)
				}
				if want, got := expPos.lineNew, comments[i].LineNew; want != got {
					t.Errorf("comment=%d, line new, want=%d got=%d", i, want, got)
				}
				if want, got := expPos.spanNew, comments[i].SpanNew; want != got {
					t.Errorf("comment=%d, span new, want=%d got=%d", i, want, got)
				}
				if want, got := expPos.lineOld, comments[i].LineOld; want != got {
					t.Errorf("comment=%d, line old, want=%d got=%d", i, want, got)
				}
				if want, got := expPos.spanOld, comments[i].SpanOld; want != got {
					t.Errorf("comment=%d, span old, want=%d got=%d", i, want, got)
				}
				if want, got := expPos.mergeBaseSHA, comments[i].MergeBaseSHA; want != got {
					t.Errorf("comment=%d, merge base sha, want=%s got=%s", i, want, got)
				}
				if want, got := expPos.sourceSHA, comments[i].SourceSHA; want != got {
					t.Errorf("comment=%d, source sha, want=%s got=%s", i, want, got)
				}
			}
		})
	}
}

type testHunkHeaderFetcher struct {
	fileName string
	headers  []git.HunkHeader
}

func (f testHunkHeaderFetcher) GetDiffHunkHeaders(
	_ context.Context,
	_ git.GetDiffHunkHeadersParams,
) (git.GetDiffHunkHeadersOutput, error) {
	return git.GetDiffHunkHeadersOutput{
		Files: []git.DiffFileHunkHeaders{
			{
				FileHeader: git.DiffFileHeader{
					OldName:    f.fileName,
					NewName:    f.fileName,
					Extensions: nil,
				},
				HunkHeaders: f.headers,
			},
		},
	}, nil
}

func TestProcessCodeComment(t *testing.T) {
	// the code comment tested in this unit test spans five lines, from line 20 to line 24
	const ccStart = 20
	const ccEnd = 24
	tests := []struct {
		name         string
		hunk         git.HunkHeader
		expOutdated  bool
		expMoveDelta int
	}{
		// only added lines
		{
			name:        "three-lines-added-before-far",
			hunk:        git.HunkHeader{OldLine: 10, OldSpan: 0, NewLine: 11, NewSpan: 3},
			expOutdated: false, expMoveDelta: 3,
		},
		{
			name:        "three-lines-added-before-but-touching",
			hunk:        git.HunkHeader{OldLine: 19, OldSpan: 0, NewLine: 20, NewSpan: 3},
			expOutdated: false, expMoveDelta: 3,
		},
		{
			name:        "three-lines-added-overlap-at-start",
			hunk:        git.HunkHeader{OldLine: 20, OldSpan: 0, NewLine: 21, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-added-inside",
			hunk:        git.HunkHeader{OldLine: 21, OldSpan: 0, NewLine: 22, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-added-overlap-at-end",
			hunk:        git.HunkHeader{OldLine: 23, OldSpan: 0, NewLine: 24, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-added-after-but-touching",
			hunk:        git.HunkHeader{OldLine: 24, OldSpan: 0, NewLine: 25, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-added-after-far",
			hunk:        git.HunkHeader{OldLine: 30, OldSpan: 0, NewLine: 31, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		// only removed lines
		{
			name:        "three-lines-removed-before-far",
			hunk:        git.HunkHeader{OldLine: 10, OldSpan: 3, NewLine: 9, NewSpan: 0},
			expOutdated: false, expMoveDelta: -3,
		},
		{
			name:        "three-lines-removed-before-but-touching",
			hunk:        git.HunkHeader{OldLine: 17, OldSpan: 3, NewLine: 16, NewSpan: 0},
			expOutdated: false, expMoveDelta: -3,
		},
		{
			name:        "three-lines-removed-overlap-at-start",
			hunk:        git.HunkHeader{OldLine: 18, OldSpan: 3, NewLine: 17, NewSpan: 0},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-removed-inside",
			hunk:        git.HunkHeader{OldLine: 21, OldSpan: 3, NewLine: 20, NewSpan: 0},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-removed-overlap-at-end",
			hunk:        git.HunkHeader{OldLine: 24, OldSpan: 3, NewLine: 23, NewSpan: 0},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-removed-after-but-touching",
			hunk:        git.HunkHeader{OldLine: 25, OldSpan: 3, NewLine: 24, NewSpan: 0},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-removed-after-far",
			hunk:        git.HunkHeader{OldLine: 30, OldSpan: 3, NewLine: 29, NewSpan: 0},
			expOutdated: false, expMoveDelta: 0,
		},
		// only changed lines
		{
			name:        "three-lines-changed-before-far",
			hunk:        git.HunkHeader{OldLine: 10, OldSpan: 3, NewLine: 10, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-before-but-touching",
			hunk:        git.HunkHeader{OldLine: 17, OldSpan: 3, NewLine: 17, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-overlap-at-start",
			hunk:        git.HunkHeader{OldLine: 18, OldSpan: 3, NewLine: 18, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-inside",
			hunk:        git.HunkHeader{OldLine: 21, OldSpan: 3, NewLine: 21, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-overlap-at-end",
			hunk:        git.HunkHeader{OldLine: 24, OldSpan: 3, NewLine: 24, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-after-but-touching",
			hunk:        git.HunkHeader{OldLine: 25, OldSpan: 3, NewLine: 25, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-after-far",
			hunk:        git.HunkHeader{OldLine: 30, OldSpan: 3, NewLine: 30, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		// mixed tests
		{
			name:        "two-lines-added-one-changed-just-before",
			hunk:        git.HunkHeader{OldLine: 19, OldSpan: 1, NewLine: 19, NewSpan: 3},
			expOutdated: false, expMoveDelta: 2,
		},
		{
			name:        "two-lines-removed-one-added-just-after",
			hunk:        git.HunkHeader{OldLine: 25, OldSpan: 2, NewLine: 25, NewSpan: 1},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "twenty-lines-added-at-line-15",
			hunk:        git.HunkHeader{OldLine: 14, OldSpan: 0, NewLine: 15, NewSpan: 20},
			expOutdated: false, expMoveDelta: 20,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outdated, moveDelta := processCodeComment(ccStart, ccEnd, test.hunk)

			if want, got := test.expOutdated, outdated; want != got {
				t.Errorf("outdated mismatch; want=%t got=%t", want, got)
				return
			}

			if want, got := test.expMoveDelta, moveDelta; want != got {
				t.Errorf("moveDelta mismatch; want=%d got=%d", want, got)
			}
		})
	}
}
