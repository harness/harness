// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package parser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/harness/gitness/gitrpc/internal/types"
)

//nolint:gocognit // it's a unit test!!!
func TestDiffCut(t *testing.T) {
	const input = `diff --git a/test.txt b/test.txt
--- a/test.txt
+++ b/test.txt
@@ -1,15 +1,11 @@
+0
 1
 2
 3
 4
 5
-6
-7
-8
+6,7,8
 9
 10
 11
 12
-13
-14
-15
`

	tests := []struct {
		name         string
		params       types.DiffCutParams
		expCutHeader string
		expCut       []string
		expError     error
	}{
		{
			name: "at-'+6,7,8':new",
			params: types.DiffCutParams{
				LineStart: 7, LineStartNew: true,
				LineEnd: 7, LineEndNew: true,
				BeforeLines: 0, AfterLines: 0,
				LineLimit: 1000,
			},
			expCutHeader: "@@ -6,3 +7 @@",
			expCut:       []string{"-6", "-7", "-8", "+6,7,8"},
			expError:     nil,
		},
		{
			name: "at-'+6,7,8':new-with-lines-around",
			params: types.DiffCutParams{
				LineStart: 7, LineStartNew: true,
				LineEnd: 7, LineEndNew: true,
				BeforeLines: 1, AfterLines: 2,
				LineLimit: 1000,
			},
			expCutHeader: "@@ -5,6 +6,4 @@",
			expCut:       []string{" 5", "-6", "-7", "-8", "+6,7,8", " 9", " 10"},
			expError:     nil,
		},
		{
			name: "at-'+0':new-with-lines-around",
			params: types.DiffCutParams{
				LineStart: 1, LineStartNew: true,
				LineEnd: 1, LineEndNew: true,
				BeforeLines: 3, AfterLines: 3,
				LineLimit: 1000,
			},
			expCutHeader: "@@ -1,3 +1,4 @@",
			expCut:       []string{"+0", " 1", " 2", " 3"},
			expError:     nil,
		},
		{
			name: "at-'-13':one-with-lines-around",
			params: types.DiffCutParams{
				LineStart: 13, LineStartNew: false,
				LineEnd: 13, LineEndNew: false,
				BeforeLines: 1, AfterLines: 1,
				LineLimit: 1000,
			},
			expCutHeader: "@@ -12,3 +11 @@",
			expCut:       []string{" 12", "-13", "-14"},
			expError:     nil,
		},
		{
			name: "at-'-13':mixed",
			params: types.DiffCutParams{
				LineStart: 7, LineStartNew: false,
				LineEnd: 7, LineEndNew: true,
				BeforeLines: 0, AfterLines: 0,
				LineLimit: 1000,
			},
			expCutHeader: "@@ -7,2 +7 @@",
			expCut:       []string{"-7", "-8", "+6,7,8"},
			expError:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hunkHeader, linesHunk, err := DiffCut(
				strings.NewReader(input),
				test.params,
			)

			//nolint:errorlint // this error will not be wrapped
			if want, got := test.expError, err; want != got {
				t.Errorf("error mismatch: want=%v got=%v", want, got)
				return
			}

			if err != nil {
				return
			}

			if test.params.LineStartNew && test.params.LineStart != hunkHeader.NewLine {
				t.Errorf("hunk line start mismatch: want=%d got=%d", test.params.LineStart, hunkHeader.NewLine)
			}

			if !test.params.LineStartNew && test.params.LineStart != hunkHeader.OldLine {
				t.Errorf("hunk line start mismatch: want=%d got=%d", test.params.LineStart, hunkHeader.OldLine)
			}

			if want, got := test.expCutHeader, linesHunk.String(); want != got {
				t.Errorf("header mismatch: want=%s got=%s", want, got)
			}

			if want, got := test.expCut, linesHunk.Lines; !reflect.DeepEqual(want, got) {
				t.Errorf("lines mismatch: want=%s got=%s", want, got)
			}
		})
	}
}

func TestStrCircBuf(t *testing.T) {
	tests := []struct {
		name string
		cap  int
		feed []string
		exp  []string
	}{
		{name: "empty", cap: 10, feed: nil, exp: []string{}},
		{name: "zero-cap", cap: 0, feed: []string{"A", "B"}, exp: []string{}},
		{name: "one", cap: 5, feed: []string{"A"}, exp: []string{"A"}},
		{name: "two", cap: 3, feed: []string{"A", "B"}, exp: []string{"A", "B"}},
		{name: "cap", cap: 3, feed: []string{"A", "B", "C"}, exp: []string{"A", "B", "C"}},
		{name: "cap+1", cap: 3, feed: []string{"A", "B", "C", "D"}, exp: []string{"B", "C", "D"}},
		{name: "cap+2", cap: 3, feed: []string{"A", "B", "C", "D", "E"}, exp: []string{"C", "D", "E"}},
		{name: "cap*2+1", cap: 2, feed: []string{"A", "B", "C", "D", "E"}, exp: []string{"D", "E"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := newStrCircBuf(test.cap)
			for _, s := range test.feed {
				b.push(s)
			}

			if want, got := test.exp, b.lines(); !reflect.DeepEqual(want, got) {
				t.Errorf("want=%v, got=%v", want, got)
			}
		})
	}
}
