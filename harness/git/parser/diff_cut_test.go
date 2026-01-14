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

package parser

import (
	"reflect"
	"strings"
	"testing"
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
		params       DiffCutParams
		expCutHeader string
		expCut       []string
		expError     error
	}{
		{
			name: "at-'+6,7,8':new",
			params: DiffCutParams{
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
			params: DiffCutParams{
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
			params: DiffCutParams{
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
			params: DiffCutParams{
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
			params: DiffCutParams{
				LineStart: 7, LineStartNew: false,
				LineEnd: 7, LineEndNew: true,
				BeforeLines: 0, AfterLines: 0,
				LineLimit: 0,
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

func TestDiffCutNoEOLInOld(t *testing.T) {
	const input = `diff --git a/test.txt b/test.txt
index 541cb64f..047d7ee2 100644
--- a/test.txt
+++ b/test.txt
@@ -1 +1,4 @@
-test
\ No newline at end of file
+123
+456
+789
`

	hh, h, err := DiffCut(
		strings.NewReader(input),
		DiffCutParams{
			LineStart:    3,
			LineStartNew: true,
			LineEnd:      3,
			LineEndNew:   true,
			BeforeLines:  1,
			AfterLines:   1,
			LineLimit:    100,
		},
	)
	if err != nil {
		t.Errorf("got error: %v", err)
		return
	}

	expectedHH := HunkHeader{OldLine: 2, OldSpan: 0, NewLine: 3, NewSpan: 1}
	if expectedHH != hh {
		t.Errorf("expected hunk header: %+v, but got: %+v", expectedHH, hh)
	}

	expectedHunkLines := Hunk{
		HunkHeader: HunkHeader{OldLine: 2, OldSpan: 0, NewLine: 2, NewSpan: 2},
		Lines:      []string{"+456", "+789"},
	}
	if !reflect.DeepEqual(expectedHunkLines, h) {
		t.Errorf("expected hunk header: %+v, but got: %+v", expectedHunkLines, h)
	}
}

func TestDiffCutNoEOLInNew(t *testing.T) {
	const input = `diff --git a/test.txt b/test.txt
index af7864ba..541cb64f 100644
--- a/test.txt
+++ b/test.txt
@@ -1,3 +1 @@
-123
-456
-789
+test
\ No newline at end of file
`
	hh, h, err := DiffCut(
		strings.NewReader(input),
		DiffCutParams{
			LineStart:    1,
			LineStartNew: true,
			LineEnd:      1,
			LineEndNew:   true,
			BeforeLines:  0,
			AfterLines:   0,
			LineLimit:    100,
		},
	)
	if err != nil {
		t.Errorf("got error: %v", err)
		return
	}

	expectedHH := HunkHeader{OldLine: 1, OldSpan: 3, NewLine: 1, NewSpan: 1}
	if expectedHH != hh {
		t.Errorf("expected hunk header: %+v, but got: %+v", expectedHH, hh)
	}

	expectedHunkLines := Hunk{
		HunkHeader: HunkHeader{OldLine: 1, OldSpan: 3, NewLine: 1, NewSpan: 1},
		Lines:      []string{"-123", "-456", "-789", "+test"},
	}
	if !reflect.DeepEqual(expectedHunkLines, h) {
		t.Errorf("expected hunk header: %+v, but got: %+v", expectedHunkLines, h)
	}
}

func TestBlobCut(t *testing.T) {
	const input = `1
2
3
4
5
6`

	tests := []struct {
		name         string
		params       DiffCutParams
		expCutHeader CutHeader
		expCut       Cut
		expError     error
	}{
		{
			name:         "first 3 lines",
			params:       DiffCutParams{LineStart: 1, LineEnd: 3, LineLimit: 40},
			expCutHeader: CutHeader{Line: 1, Span: 3},
			expCut:       Cut{CutHeader: CutHeader{Line: 1, Span: 3}, Lines: []string{"1", "2", "3"}},
		},
		{
			name:         "last 2 lines",
			params:       DiffCutParams{LineStart: 5, LineEnd: 6, LineLimit: 40},
			expCutHeader: CutHeader{Line: 5, Span: 2},
			expCut:       Cut{CutHeader: CutHeader{Line: 5, Span: 2}, Lines: []string{"5", "6"}},
		},
		{
			name:         "first 3 lines with 1 more",
			params:       DiffCutParams{LineStart: 1, LineEnd: 3, BeforeLines: 1, AfterLines: 1, LineLimit: 40},
			expCutHeader: CutHeader{Line: 1, Span: 3},
			expCut:       Cut{CutHeader: CutHeader{Line: 1, Span: 4}, Lines: []string{"1", "2", "3", "4"}},
		},
		{
			name:         "last 2 lines with 2 more",
			params:       DiffCutParams{LineStart: 5, LineEnd: 6, BeforeLines: 2, AfterLines: 2, LineLimit: 40},
			expCutHeader: CutHeader{Line: 5, Span: 2},
			expCut:       Cut{CutHeader: CutHeader{Line: 3, Span: 4}, Lines: []string{"3", "4", "5", "6"}},
		},
		{
			name:         "mid range",
			params:       DiffCutParams{LineStart: 2, LineEnd: 4, BeforeLines: 100, AfterLines: 100, LineLimit: 40},
			expCutHeader: CutHeader{Line: 2, Span: 3},
			expCut:       Cut{CutHeader: CutHeader{Line: 1, Span: 6}, Lines: []string{"1", "2", "3", "4", "5", "6"}},
		},
		{
			name:     "out of range 1",
			params:   DiffCutParams{LineStart: 15, LineEnd: 17, LineLimit: 40},
			expError: ErrHunkNotFound,
		},
		{
			name:     "out of range 2",
			params:   DiffCutParams{LineStart: 5, LineEnd: 7, BeforeLines: 1, AfterLines: 1, LineLimit: 40},
			expError: ErrHunkNotFound,
		},
		{
			name:         "limited",
			params:       DiffCutParams{LineStart: 3, LineEnd: 4, BeforeLines: 1, AfterLines: 1, LineLimit: 3},
			expCutHeader: CutHeader{Line: 3, Span: 2},
			expCut:       Cut{CutHeader: CutHeader{Line: 2, Span: 3}, Lines: []string{"2", "3", "4"}},
		},
		{
			name:         "unlimited",
			params:       DiffCutParams{LineStart: 1, LineEnd: 6, BeforeLines: 1, AfterLines: 1, LineLimit: 0},
			expCutHeader: CutHeader{Line: 1, Span: 6},
			expCut:       Cut{CutHeader: CutHeader{Line: 1, Span: 6}, Lines: []string{"1", "2", "3", "4", "5", "6"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ch, c, err := BlobCut(strings.NewReader(input), test.params)
			if err != test.expError { //nolint:errorlint // it's a unit test and no errors are wrapped
				t.Errorf("test failed with error: %s", err.Error())
				return
			}

			if want, got := test.expCutHeader, ch; want != got {
				t.Errorf("cut header: want=%+v got: %+v", want, got)
			}

			if want, got := test.expCut, c; !reflect.DeepEqual(want, got) {
				t.Errorf("cut: want=%+v got: %+v", want, got)
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
