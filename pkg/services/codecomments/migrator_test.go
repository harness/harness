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
	"testing"

	"github.com/harness/gitness/gitrpc"
)

func TestProcessCodeComment(t *testing.T) {
	// the code comment tested in this unit test spans five lines, from line 20 to line 24
	const ccStart = 20
	const ccEnd = 24
	tests := []struct {
		name         string
		hunk         gitrpc.HunkHeader
		expOutdated  bool
		expMoveDelta int
	}{
		// only added lines
		{
			name:        "three-lines-added-before-far",
			hunk:        gitrpc.HunkHeader{OldLine: 10, OldSpan: 0, NewLine: 11, NewSpan: 3},
			expOutdated: false, expMoveDelta: 3,
		},
		{
			name:        "three-lines-added-before-but-touching",
			hunk:        gitrpc.HunkHeader{OldLine: 19, OldSpan: 0, NewLine: 20, NewSpan: 3},
			expOutdated: false, expMoveDelta: 3,
		},
		{
			name:        "three-lines-added-overlap-at-start",
			hunk:        gitrpc.HunkHeader{OldLine: 20, OldSpan: 0, NewLine: 21, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-added-inside",
			hunk:        gitrpc.HunkHeader{OldLine: 21, OldSpan: 0, NewLine: 22, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-added-overlap-at-end",
			hunk:        gitrpc.HunkHeader{OldLine: 23, OldSpan: 0, NewLine: 24, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-added-after-but-touching",
			hunk:        gitrpc.HunkHeader{OldLine: 24, OldSpan: 0, NewLine: 25, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-added-after-far",
			hunk:        gitrpc.HunkHeader{OldLine: 30, OldSpan: 0, NewLine: 31, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		// only removed lines
		{
			name:        "three-lines-removed-before-far",
			hunk:        gitrpc.HunkHeader{OldLine: 10, OldSpan: 3, NewLine: 9, NewSpan: 0},
			expOutdated: false, expMoveDelta: -3,
		},
		{
			name:        "three-lines-removed-before-but-touching",
			hunk:        gitrpc.HunkHeader{OldLine: 17, OldSpan: 3, NewLine: 16, NewSpan: 0},
			expOutdated: false, expMoveDelta: -3,
		},
		{
			name:        "three-lines-removed-overlap-at-start",
			hunk:        gitrpc.HunkHeader{OldLine: 18, OldSpan: 3, NewLine: 17, NewSpan: 0},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-removed-inside",
			hunk:        gitrpc.HunkHeader{OldLine: 21, OldSpan: 3, NewLine: 20, NewSpan: 0},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-removed-overlap-at-end",
			hunk:        gitrpc.HunkHeader{OldLine: 24, OldSpan: 3, NewLine: 23, NewSpan: 0},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-removed-after-but-touching",
			hunk:        gitrpc.HunkHeader{OldLine: 25, OldSpan: 3, NewLine: 24, NewSpan: 0},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-removed-after-far",
			hunk:        gitrpc.HunkHeader{OldLine: 30, OldSpan: 3, NewLine: 29, NewSpan: 0},
			expOutdated: false, expMoveDelta: 0,
		},
		// only changed lines
		{
			name:        "three-lines-changed-before-far",
			hunk:        gitrpc.HunkHeader{OldLine: 10, OldSpan: 3, NewLine: 10, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-before-but-touching",
			hunk:        gitrpc.HunkHeader{OldLine: 17, OldSpan: 3, NewLine: 17, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-overlap-at-start",
			hunk:        gitrpc.HunkHeader{OldLine: 18, OldSpan: 3, NewLine: 18, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-inside",
			hunk:        gitrpc.HunkHeader{OldLine: 21, OldSpan: 3, NewLine: 21, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-overlap-at-end",
			hunk:        gitrpc.HunkHeader{OldLine: 24, OldSpan: 3, NewLine: 24, NewSpan: 3},
			expOutdated: true, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-after-but-touching",
			hunk:        gitrpc.HunkHeader{OldLine: 25, OldSpan: 3, NewLine: 25, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "three-lines-changed-after-far",
			hunk:        gitrpc.HunkHeader{OldLine: 30, OldSpan: 3, NewLine: 30, NewSpan: 3},
			expOutdated: false, expMoveDelta: 0,
		},
		// mixed tests
		{
			name:        "two-lines-added-one-changed-just-before",
			hunk:        gitrpc.HunkHeader{OldLine: 19, OldSpan: 1, NewLine: 19, NewSpan: 3},
			expOutdated: false, expMoveDelta: 2,
		},
		{
			name:        "two-lines-removed-one-added-just-after",
			hunk:        gitrpc.HunkHeader{OldLine: 25, OldSpan: 2, NewLine: 25, NewSpan: 1},
			expOutdated: false, expMoveDelta: 0,
		},
		{
			name:        "twenty-lines-added-at-line-15",
			hunk:        gitrpc.HunkHeader{OldLine: 14, OldSpan: 0, NewLine: 15, NewSpan: 20},
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
