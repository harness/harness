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
	"bufio"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/git/enum"

	"github.com/google/go-cmp/cmp"
)

func TestGetHunkHeaders(t *testing.T) {
	input := `diff --git a/new_file.txt b/new_file.txt
new file mode 100644
index 0000000..fb0c863
--- /dev/null
+++ b/new_file.txt
@@ -0,0 +1,3 @@
+This is a new file
+created for this
+unit test.
diff --git a/old_file_name.txt b/changed_file.txt
index f043b93..e9449b5 100644
--- a/changed_file.txt
+++ b/changed_file.txt
@@ -7,3 +7,4 @@
 Unchanged line 
-Removed line 1
+Added line 1
+Added line 2
 Unchanged line 
@@ -27,2 +28,3 @@
 Unchanged line 
+Added line
 Unchanged line 
diff --git a/deleted_file.txt b/deleted_file.txt
deleted file mode 100644
index f043b93..0000000
--- a/deleted_file.txt
+++ /dev/null
@@ -1,3 +0,0 @@
-This is content of
-a deleted file
-in git diff output.
`

	got, err := GetHunkHeaders(strings.NewReader(input))
	if err != nil {
		t.Errorf("got error: %v", err)
		return
	}

	want := []*DiffFileHunkHeaders{
		{
			FileHeader: DiffFileHeader{
				OldFileName: "new_file.txt",
				NewFileName: "new_file.txt",
				Extensions: map[string]string{
					enum.DiffExtHeaderNewFileMode: "100644",
					enum.DiffExtHeaderIndex:       "0000000..fb0c863",
				},
			},
			HunksHeaders: []HunkHeader{{OldLine: 0, OldSpan: 0, NewLine: 1, NewSpan: 3}},
		},
		{
			FileHeader: DiffFileHeader{
				OldFileName: "old_file_name.txt",
				NewFileName: "changed_file.txt",
				Extensions: map[string]string{
					enum.DiffExtHeaderIndex: "f043b93..e9449b5 100644",
				},
			},
			HunksHeaders: []HunkHeader{
				{OldLine: 7, OldSpan: 3, NewLine: 7, NewSpan: 4},
				{OldLine: 27, OldSpan: 2, NewLine: 28, NewSpan: 3},
			},
		},
		{
			FileHeader: DiffFileHeader{
				OldFileName: "deleted_file.txt",
				NewFileName: "deleted_file.txt",
				Extensions: map[string]string{
					enum.DiffExtHeaderDeletedFileMode: "100644",
					enum.DiffExtHeaderIndex:           "f043b93..0000000",
				},
			},
			HunksHeaders: []HunkHeader{{OldLine: 1, OldSpan: 3, NewLine: 0, NewSpan: 0}},
		},
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}

func TestReadLinePrefix(t *testing.T) {
	const maxLen = 256
	tests := []struct {
		name    string
		wf      func(w io.Writer)
		expLens []int
	}{
		{
			name:    "empty",
			wf:      func(io.Writer) {},
			expLens: nil,
		},
		{
			name: "single",
			wf: func(w io.Writer) {
				_, _ = w.Write([]byte("aaa"))
			},
			expLens: []int{3},
		},
		{
			name: "single-eol",
			wf: func(w io.Writer) {
				_, _ = w.Write([]byte("aaa\n"))
			},
			expLens: []int{3},
		},
		{
			name: "two-lines",
			wf: func(w io.Writer) {
				_, _ = w.Write([]byte("aa\nbb"))
			},
			expLens: []int{2, 2},
		},
		{
			name: "two-lines-crlf",
			wf: func(w io.Writer) {
				_, _ = w.Write([]byte("aa\r\nbb\r\n"))
			},
			expLens: []int{2, 2},
		},
		{
			name: "empty-line",
			wf: func(w io.Writer) {
				_, _ = w.Write([]byte("aa\n\ncc"))
			},
			expLens: []int{2, 0, 2},
		},
		{
			name: "too-long",
			wf: func(w io.Writer) {
				for i := 0; i < maxLen; i++ {
					_, _ = w.Write([]byte("a"))
				}
				_, _ = w.Write([]byte("\n"))
				for i := 0; i < maxLen*2; i++ {
					_, _ = w.Write([]byte("b"))
				}
				_, _ = w.Write([]byte("\n"))
				for i := 0; i < maxLen/2; i++ {
					_, _ = w.Write([]byte("c"))
				}
				_, _ = w.Write([]byte("\n"))
			},
			expLens: []int{maxLen, maxLen, maxLen / 2},
		},
		{
			name: "overflow-buffer",
			wf: func(w io.Writer) {
				for i := 0; i < bufio.MaxScanTokenSize+1; i++ {
					_, _ = w.Write([]byte("a"))
				}
				_, _ = w.Write([]byte("\n"))
				for i := 0; i < bufio.MaxScanTokenSize*2; i++ {
					_, _ = w.Write([]byte("b"))
				}
				_, _ = w.Write([]byte("\n"))
				for i := 0; i < bufio.MaxScanTokenSize; i++ {
					_, _ = w.Write([]byte("c"))
				}
			},
			expLens: []int{maxLen, maxLen, maxLen},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pr, pw := io.Pipe()
			defer pr.Close()

			go func() {
				test.wf(pw)
				_ = pw.Close()
			}()

			br := bufio.NewReader(pr)

			for i, expLen := range test.expLens {
				expLine := strings.Repeat(string(rune('a'+i)), expLen)
				line, err := readLinePrefix(br, maxLen)
				if err != nil && err != io.EOF { //nolint:errorlint
					t.Errorf("got error: %s", err.Error())
					return
				}
				if want, got := expLine, line; want != got {
					t.Errorf("line %d mismatch want=%s got=%s", i, want, got)
					return
				}
			}

			line, err := readLinePrefix(br, maxLen)
			if line != "" || err != io.EOF { //nolint:errorlint
				t.Errorf("expected empty line and EOF but got: line=%s err=%v", line, err)
			}
		})
	}
}
