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
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/harness/gitness/git/parser"

	"github.com/google/go-cmp/cmp"
)

func Test_modifyHeader(t *testing.T) {
	type args struct {
		hunk      parser.HunkHeader
		startLine int
		endLine   int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "test empty",
			args: args{
				hunk: parser.HunkHeader{
					OldLine: 0,
					OldSpan: 0,
					NewLine: 0,
					NewSpan: 0,
				},
				startLine: 2,
				endLine:   10,
			},
			want: []byte("@@ -0,0 +0,0 @@"),
		},
		{
			name: "test empty 1",
			args: args{
				hunk: parser.HunkHeader{
					OldLine: 0,
					OldSpan: 0,
					NewLine: 0,
					NewSpan: 0,
				},
				startLine: 0,
				endLine:   0,
			},
			want: []byte("@@ -0,0 +0,0 @@"),
		},
		{
			name: "test empty old",
			args: args{
				hunk: parser.HunkHeader{
					OldLine: 0,
					OldSpan: 0,
					NewLine: 1,
					NewSpan: 10,
				},
				startLine: 2,
				endLine:   10,
			},
			want: []byte("@@ -0,0 +2,8 @@"),
		},
		{
			name: "test empty new",
			args: args{
				hunk: parser.HunkHeader{
					OldLine: 1,
					OldSpan: 10,
					NewLine: 0,
					NewSpan: 0,
				},
				startLine: 2,
				endLine:   10,
			},
			want: []byte("@@ -2,8 +0,0 @@"),
		},
		{
			name: "test 1",
			args: args{
				hunk: parser.HunkHeader{
					OldLine: 2,
					OldSpan: 20,
					NewLine: 2,
					NewSpan: 20,
				},
				startLine: 5,
				endLine:   10,
			},
			want: []byte("@@ -5,5 +5,5 @@"),
		},
		{
			name: "test 2",
			args: args{
				hunk: parser.HunkHeader{
					OldLine: 2,
					OldSpan: 20,
					NewLine: 2,
					NewSpan: 20,
				},
				startLine: 15,
				endLine:   25,
			},
			want: []byte("@@ -15,7 +15,7 @@"),
		},
		{
			name: "test 4",
			args: args{
				hunk: parser.HunkHeader{
					OldLine: 1,
					OldSpan: 10,
					NewLine: 1,
					NewSpan: 10,
				},
				startLine: 15,
				endLine:   20,
			},
			want: []byte("@@ -1,10 +1,10 @@"),
		},
		{
			name: "test 5",
			args: args{
				hunk: parser.HunkHeader{
					OldLine: 1,
					OldSpan: 108,
					NewLine: 1,
					NewSpan: 108,
				},
				startLine: 5,
				endLine:   0,
			},
			want: []byte("@@ -5,108 +5,108 @@"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := modifyHeader(tt.args.hunk, tt.args.startLine, tt.args.endLine); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("modifyHeader() = %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_cutLinesFromFullDiff(t *testing.T) {
	type args struct {
		r         io.Reader
		startLine int
		endLine   int
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "test empty",
			args: args{
				r: strings.NewReader(`diff --git a/file.txt b/file.txt
index f0eec86f614944a81f87d879ebdc9a79aea0d7ea..47d2739ba2c34690248c8f91b84bb54e8936899a 100644
--- a/file.txt
+++ b/file.txt
@@ -0,0 +0,0 @@
`),
				startLine: 2,
				endLine:   10,
			},
			wantW: `diff --git a/file.txt b/file.txt
index f0eec86f614944a81f87d879ebdc9a79aea0d7ea..47d2739ba2c34690248c8f91b84bb54e8936899a 100644
--- a/file.txt
+++ b/file.txt
@@ -0,0 +0,0 @@
`,
		},
		{
			name: "test 1",
			args: args{
				r: strings.NewReader(`diff --git a/file.txt b/file.txt
index f0eec86f614944a81f87d879ebdc9a79aea0d7ea..47d2739ba2c34690248c8f91b84bb54e8936899a 100644
--- a/file.txt
+++ b/file.txt
@@ -1,9 +1,9 @@
some content
some content
some content
-some content
+some content
some content
some content
some content
some content
some content
`),
				startLine: 2,
				endLine:   10,
			},
			wantW: `diff --git a/file.txt b/file.txt
index f0eec86f614944a81f87d879ebdc9a79aea0d7ea..47d2739ba2c34690248c8f91b84bb54e8936899a 100644
--- a/file.txt
+++ b/file.txt
@@ -2,8 +2,8 @@
some content
some content
-some content
+some content
some content
some content
some content
some content
some content
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			err := cutLinesFromFullFileDiff(w, tt.args.r, tt.args.startLine, tt.args.endLine)
			if (err != nil) != tt.wantErr {
				t.Errorf("cutLinesFromFullFileDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("cutLinesFromFullFileDiff() gotW = %v, want %v, diff: %s", gotW, tt.wantW, cmp.Diff(gotW, tt.wantW))
			}
		})
	}
}
