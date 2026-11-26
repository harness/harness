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

package parser

import (
	"bufio"
	"errors"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gtxDiffPrefix = "diff --git a/f.txt b/f.txt\nindex 1111111..2222222 100644\n--- a/f.txt\n+++ b/f.txt\n"

func TestGtxDiffCut(t *testing.T) {
	hunk := gtxDiffPrefix +
		"@@ -1,6 +1,7 @@\n" +
		" l1\n" +
		" l2\n" +
		"-l3\n" +
		"+L3\n" +
		"+L3b\n" +
		"\\ No newline at end of file\n" +
		" l4\n" +
		" l5\n" +
		" l6\n"

	tests := []struct {
		name       string
		input      string
		params     DiffCutParams
		wantHeader HunkHeader
		wantHunk   Hunk
		wantErr    error
	}{
		{
			name:       "old side range",
			input:      hunk,
			params:     DiffCutParams{LineStart: 2, LineEnd: 3},
			wantHeader: HunkHeader{OldLine: 2, OldSpan: 2, NewLine: 2, NewSpan: 1},
			wantHunk: Hunk{
				HunkHeader: HunkHeader{OldLine: 2, OldSpan: 2, NewLine: 2, NewSpan: 1},
				Lines:      []string{" l2", "-l3"},
			},
		},
		{
			name:  "new side range with context",
			input: hunk,
			params: DiffCutParams{
				LineStart: 3, LineStartNew: true, LineEnd: 4, LineEndNew: true, BeforeLines: 1, AfterLines: 1,
			},
			wantHeader: HunkHeader{OldLine: 3, OldSpan: 1, NewLine: 3, NewSpan: 2},
			wantHunk: Hunk{
				HunkHeader: HunkHeader{OldLine: 2, OldSpan: 3, NewLine: 2, NewSpan: 4},
				Lines:      []string{" l2", "-l3", "+L3", "+L3b", " l4"},
			},
		},
		{
			name:       "range until the end of the hunk",
			input:      hunk,
			params:     DiffCutParams{LineStart: 5, LineEnd: 100, AfterLines: 3},
			wantHeader: HunkHeader{OldLine: 5, OldSpan: 2, NewLine: 6, NewSpan: 2},
			wantHunk: Hunk{
				HunkHeader: HunkHeader{OldLine: 5, OldSpan: 2, NewLine: 6, NewSpan: 2},
				Lines:      []string{" l5", " l6"},
			},
		},
		{
			name:       "line limit",
			input:      hunk,
			params:     DiffCutParams{LineStart: 1, LineEnd: 100, LineLimit: 2},
			wantHeader: HunkHeader{OldLine: 1, OldSpan: 2, NewLine: 1, NewSpan: 2},
			wantHunk: Hunk{
				HunkHeader: HunkHeader{OldLine: 1, OldSpan: 2, NewLine: 1, NewSpan: 2},
				Lines:      []string{" l1", " l2"},
			},
		},
		{
			name: "range beyond the hunk", input: hunk,
			params: DiffCutParams{LineStart: 50, LineEnd: 60}, wantErr: ErrHunkNotFound,
		},
		{
			name: "no file header", input: "nothing here\n",
			params: DiffCutParams{LineStart: 1, LineEnd: 1}, wantErr: ErrHunkNotFound,
		},
		{
			name: "no hunk header", input: gtxDiffPrefix,
			params: DiffCutParams{LineStart: 1, LineEnd: 1}, wantErr: ErrHunkNotFound,
		},
		{
			name:    "empty line in hunk",
			input:   gtxDiffPrefix + "@@ -1,2 +1,2 @@\n l1\n\n l2\n",
			params:  DiffCutParams{LineStart: 1, LineEnd: 2},
			wantErr: ErrHunkNotFound,
		},
		{
			name:       "hunk ended by next file",
			input:      gtxDiffPrefix + "@@ -1,1 +1,1 @@\n l1\ndiff --git a/g b/g\n",
			params:     DiffCutParams{LineStart: 1, LineEnd: 1, AfterLines: 2},
			wantHeader: HunkHeader{OldLine: 1, OldSpan: 1, NewLine: 1, NewSpan: 1},
			wantHunk: Hunk{
				HunkHeader: HunkHeader{OldLine: 1, OldSpan: 1, NewLine: 1, NewSpan: 1},
				Lines:      []string{" l1"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header, h, err := DiffCut(strings.NewReader(test.input), test.params)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantHeader, header)
			assert.Equal(t, test.wantHunk, h)
		})
	}
}

func TestGtxDiffCut_ReadErrors(t *testing.T) {
	readErr := errors.New("gtx read failure")

	t.Run("while looking for file header", func(t *testing.T) {
		_, _, err := DiffCut(iotest.ErrReader(readErr), DiffCutParams{LineStart: 1, LineEnd: 1})
		require.ErrorIs(t, err, readErr)
	})

	t.Run("while looking for hunk header", func(t *testing.T) {
		r := &gtxFailAfterReader{data: []byte(gtxDiffPrefix), err: readErr}
		_, _, err := DiffCut(r, DiffCutParams{LineStart: 1, LineEnd: 1})
		require.ErrorIs(t, err, readErr)
	})

	t.Run("while reading lines", func(t *testing.T) {
		r := &gtxFailAfterReader{data: []byte(gtxDiffPrefix + "@@ -1,3 +1,3 @@\n l1\n"), err: readErr}
		_, _, err := DiffCut(r, DiffCutParams{LineStart: 1, LineEnd: 3})
		require.ErrorIs(t, err, readErr)
	})

	t.Run("while reading lines after the cut", func(t *testing.T) {
		r := &gtxFailAfterReader{data: []byte(gtxDiffPrefix + "@@ -1,3 +1,3 @@\n l1\n l2\n"), err: readErr}
		_, _, err := DiffCut(r, DiffCutParams{LineStart: 1, LineEnd: 1, AfterLines: 5})
		require.ErrorIs(t, err, readErr)
	})
}

// gtxFailAfterReader returns the data and then the error.
type gtxFailAfterReader struct {
	data []byte
	err  error
}

func (r *gtxFailAfterReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}

	n := copy(p, r.data)
	r.data = r.data[n:]

	return n, nil
}

func TestGtxBlobCut(t *testing.T) {
	content := "l1\nl2\nl3\nl4\nl5\n"

	tests := []struct {
		name       string
		input      string
		params     DiffCutParams
		wantHeader CutHeader
		wantCut    Cut
		wantErr    error
	}{
		{
			name:       "exact lines",
			input:      content,
			params:     DiffCutParams{LineStart: 2, LineEnd: 3},
			wantHeader: CutHeader{Line: 2, Span: 2},
			wantCut:    Cut{CutHeader: CutHeader{Line: 2, Span: 2}, Lines: []string{"l2", "l3"}},
		},
		{
			name:       "with context",
			input:      content,
			params:     DiffCutParams{LineStart: 2, LineEnd: 3, BeforeLines: 5, AfterLines: 1},
			wantHeader: CutHeader{Line: 2, Span: 2},
			wantCut:    Cut{CutHeader: CutHeader{Line: 1, Span: 4}, Lines: []string{"l1", "l2", "l3", "l4"}},
		},
		{
			name:       "line limit",
			input:      content,
			params:     DiffCutParams{LineStart: 1, LineEnd: 1, AfterLines: 4, LineLimit: 2},
			wantHeader: CutHeader{Line: 1, Span: 1},
			wantCut:    Cut{CutHeader: CutHeader{Line: 1, Span: 2}, Lines: []string{"l1", "l2"}},
		},
		{
			name: "beyond end of file", input: content,
			params: DiffCutParams{LineStart: 4, LineEnd: 9}, wantErr: ErrHunkNotFound,
		},
		{name: "empty file", input: "", params: DiffCutParams{LineStart: 1, LineEnd: 1}, wantErr: ErrHunkNotFound},
		{
			name: "binary file", input: "l1\n\xff\xfe\n",
			params: DiffCutParams{LineStart: 1, LineEnd: 2}, wantErr: ErrBinaryFile,
		},
		{
			name:    "line too long",
			input:   strings.Repeat("a", bufio.MaxScanTokenSize+10) + "\n",
			params:  DiffCutParams{LineStart: 1, LineEnd: 1},
			wantErr: ErrBinaryFile,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header, cut, err := BlobCut(strings.NewReader(test.input), test.params)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantHeader, header)
			assert.Equal(t, test.wantCut, cut)
		})
	}

	t.Run("read error", func(t *testing.T) {
		readErr := errors.New("gtx read failure")
		_, _, err := BlobCut(iotest.ErrReader(readErr), DiffCutParams{LineStart: 1, LineEnd: 1})
		require.ErrorIs(t, err, readErr)
	})
}

func TestGtxObjectHeaderIdentity(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantName  string
		wantEmail string
		wantUnix  int64
		wantZone  string
		wantOff   int
		wantErr   bool
	}{
		{
			name: "utc", line: "Jane Doe <jane@example.com> 1700000000 +0000",
			wantName: "Jane Doe", wantEmail: "jane@example.com", wantUnix: 1700000000, wantZone: "+0000",
		},
		{
			name: "positive offset", line: "A <a@example.com> 1700000000 +0530",
			wantName: "A", wantEmail: "a@example.com", wantUnix: 1700000000, wantZone: "+0530", wantOff: 5*3600 + 30*60,
		},
		{
			name: "negative offset", line: "A <a@example.com> 1700000000 -0800",
			wantName: "A", wantEmail: "a@example.com", wantUnix: 1700000000, wantZone: "-0800", wantOff: -8 * 3600,
		},
		{
			name: "negative offset with minutes", line: "A <a@example.com> 1700000000 -0930",
			wantName: "A", wantEmail: "a@example.com", wantUnix: 1700000000, wantZone: "-0930", wantOff: -(9*3600 + 30*60),
		},
		{
			name: "empty email", line: "Nobody <> 1700000000 +0100",
			wantName: "Nobody", wantUnix: 1700000000, wantZone: "+0100", wantOff: 3600,
		},
		{name: "no match", line: "not an identity"},
		{name: "empty", line: ""},
		{name: "unix time overflow", line: "A <a@example.com> 99999999999999999999 +0000", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, email, ts, err := ObjectHeaderIdentity(test.line)
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantName, name)
			assert.Equal(t, test.wantEmail, email)
			if test.wantZone == "" {
				assert.True(t, ts.IsZero())
				return
			}

			assert.Equal(t, test.wantUnix, ts.Unix())
			zone, off := ts.Zone()
			assert.Equal(t, test.wantZone, zone)
			assert.Equal(t, test.wantOff, off)
			assert.Equal(t, time.Unix(test.wantUnix, 0).UTC(), ts.UTC())
		})
	}
}

func TestGtxReferenceList(t *testing.T) {
	shaA := strings.Repeat("a", 40)
	shaB := strings.Repeat("b", 40)

	t.Run("references", func(t *testing.T) {
		refs, err := ReferenceList(strings.NewReader(shaA + " refs/heads/main\n" + shaB + "\trefs/tags/v1 beta\n"))
		require.NoError(t, err)
		require.Len(t, refs, 2)
		assert.Equal(t, shaA, refs["refs/heads/main"].String())
		assert.Equal(t, shaB, refs["refs/tags/v1 beta"].String())
	})

	t.Run("empty", func(t *testing.T) {
		refs, err := ReferenceList(strings.NewReader(""))
		require.NoError(t, err)
		assert.Empty(t, refs)
	})

	t.Run("unexpected line", func(t *testing.T) {
		_, err := ReferenceList(strings.NewReader("garbage\n"))
		require.ErrorContains(t, err, "unexpected output")
	})

	t.Run("invalid sha", func(t *testing.T) {
		_, err := ReferenceList(strings.NewReader("abc refs/heads/main\n"))
		require.ErrorContains(t, err, "failed to parse reference sha")
	})

	t.Run("read error", func(t *testing.T) {
		_, err := ReferenceList(iotest.ErrReader(errors.New("gtx read failure")))
		require.ErrorContains(t, err, "gtx read failure")
	})
}

func TestGtxReadTextFile(t *testing.T) {
	collect := func(t *testing.T, s Scanner) []string {
		t.Helper()
		var lines []string
		for s.Scan() {
			lines = append(lines, s.Text())
		}
		return lines
	}

	t.Run("line endings detected", func(t *testing.T) {
		s, le, err := ReadTextFile(strings.NewReader("a\r\nb\r\nc"), nil)
		require.NoError(t, err)
		assert.Equal(t, "\r\n", le)
		assert.Equal(t, []string{"a\r\n", "b\r\n", "c"}, collect(t, s))
		require.NoError(t, s.Err())
	})

	t.Run("default line ending", func(t *testing.T) {
		_, le, err := ReadTextFile(strings.NewReader("no line ending"), nil)
		require.NoError(t, err)
		assert.Equal(t, "\n", le)
	})

	t.Run("line endings overwritten", func(t *testing.T) {
		crlf := "\r\n"
		s, _, err := ReadTextFile(strings.NewReader("a\nb\r\nc"), &crlf)
		require.NoError(t, err)
		assert.Equal(t, []string{"a\r\n", "b\r\n", "c"}, collect(t, s))

		lf := "\n"
		s, _, err = ReadTextFile(strings.NewReader("a\r\nb\nc"), &lf)
		require.NoError(t, err)
		assert.Equal(t, []string{"a\n", "b\n", "c"}, collect(t, s))
	})

	t.Run("binary content", func(t *testing.T) {
		s, _, err := ReadTextFile(strings.NewReader("ok\n\xff\xfe\n"), nil)
		require.NoError(t, err)
		assert.True(t, s.Scan())
		assert.False(t, s.Scan())
		require.ErrorIs(t, s.Err(), ErrBinaryFile)
		assert.Nil(t, s.Bytes())
	})

	t.Run("line too long", func(t *testing.T) {
		s, _, err := ReadTextFile(strings.NewReader("ok\n"+strings.Repeat("a", bufio.MaxScanTokenSize+10)+"\n"), nil)
		require.NoError(t, err)
		assert.True(t, s.Scan())
		assert.False(t, s.Scan())
		require.ErrorIs(t, s.Err(), ErrBinaryFile)
	})

	t.Run("read error on first line", func(t *testing.T) {
		_, _, err := ReadTextFile(iotest.ErrReader(errors.New("gtx read failure")), nil)
		require.ErrorContains(t, err, "gtx read failure")
	})
}

func TestGtxGetHunkHeaders(t *testing.T) {
	input := "diff --git a/a.txt b/a.txt\n" +
		"index 1111111..2222222 100644\n" +
		"--- a/a.txt\n" +
		"+++ b/a.txt\n" +
		"@@ -1,2 +1,3 @@ func a()\n" +
		" x\n" +
		"+y\n" +
		"@@ -10 +11 @@\n" +
		"-z\n" +
		"+Z\n" +
		"diff --git a/b.txt b/b.txt\n" +
		"new file mode 100644\n" +
		"index 0000000..3333333\n" +
		"--- /dev/null\n" +
		"+++ b/b.txt\n" +
		"@@ -0,0 +1 @@\n" +
		"+n\n"

	files, err := GetHunkHeaders(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, files, 2)

	assert.Equal(t, "a.txt", files[0].FileHeader.NewFileName)
	require.Len(t, files[0].HunksHeaders, 2)
	assert.Equal(t, HunkHeader{OldLine: 1, OldSpan: 2, NewLine: 1, NewSpan: 3, Text: "func a()"}, files[0].HunksHeaders[0])
	assert.Equal(t, HunkHeader{OldLine: 10, OldSpan: 1, NewLine: 11, NewSpan: 1}, files[0].HunksHeaders[1])
	assert.Equal(t, "1111111..2222222 100644", files[0].FileHeader.Extensions["index"])

	assert.Equal(t, "b.txt", files[1].FileHeader.NewFileName)
	assert.Equal(t, "100644", files[1].FileHeader.Extensions["new file mode"])
	require.Len(t, files[1].HunksHeaders, 1)

	t.Run("hunk before file header", func(t *testing.T) {
		_, err := GetHunkHeaders(strings.NewReader("@@ -1 +1 @@\n"))
		require.ErrorIs(t, err, ErrHunkNotFound)
	})

	t.Run("empty", func(t *testing.T) {
		got, err := GetHunkHeaders(strings.NewReader(""))
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("read error", func(t *testing.T) {
		_, err := GetHunkHeaders(iotest.ErrReader(errors.New("gtx read failure")))
		require.ErrorContains(t, err, "gtx read failure")
	})

	t.Run("very long lines are consumed", func(t *testing.T) {
		long := "diff --git a/a b/a\n+" + strings.Repeat("x", 10000) + "\n@@ -1 +1 @@\n"
		got, err := GetHunkHeaders(strings.NewReader(long))
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Len(t, got[0].HunksHeaders, 1)
	})
}
