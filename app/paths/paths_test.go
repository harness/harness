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

package paths

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Concatenate(t *testing.T) {
	type testCase struct {
		in   []string
		want string
	}
	tests := []testCase{
		{
			in:   nil,
			want: "",
		},
		{
			in:   []string{},
			want: "",
		},
		{
			in: []string{
				"",
			},
			want: "",
		},
		{
			in: []string{
				"",
				"",
			},
			want: "",
		},
		{
			in: []string{
				"a",
			},
			want: "a",
		},
		{
			in: []string{
				"",
				"a",
			},
			want: "a",
		},
		{
			in: []string{
				"a",
				"",
			},
			want: "a",
		},
		{
			in: []string{
				"",
				"a",
				"",
			},
			want: "a",
		},
		{
			in: []string{
				"a",
				"b",
			},
			want: "a/b",
		},
		{
			in: []string{
				"a",
				"b",
				"c",
			},
			want: "a/b/c",
		},
		{
			in: []string{
				"seg1",
				"seg2/seg3",
				"seg4",
			},
			want: "seg1/seg2/seg3/seg4",
		},
		{
			in: []string{
				"/",
				"/",
				"/",
			},
			want: "",
		},
		{
			in: []string{
				"//",
			},
			want: "",
		},
		{
			in: []string{
				"/a/",
			},
			want: "a",
		},
		{
			in: []string{
				"/a/",
				"/b/",
			},
			want: "a/b",
		},
		{
			in: []string{
				"/a/b/",
				"//c//d//",
				"///e///f///",
			},
			want: "a/b/c/d/e/f",
		},
	}
	for _, tt := range tests {
		got := Concatenate(tt.in...)
		assert.Equal(t, tt.want, got, "path isn't matching for %v", tt.in)
	}
}

func Test_Depth(t *testing.T) {
	type testCase struct {
		in   string
		want int
	}
	tests := []testCase{
		{
			in:   "",
			want: 0,
		},
		{
			in:   "/",
			want: 0,
		},
		{
			in:   "a",
			want: 1,
		},
		{
			in:   "/a/",
			want: 1,
		},
		{
			in:   "a/b",
			want: 2,
		},
		{
			in:   "/a/b/c/d/e/f/",
			want: 6,
		},
	}
	for _, tt := range tests {
		got := Depth(tt.in)
		assert.Equal(t, tt.want, got, "depth isn't matching for %q", tt.in)
	}
}

func Test_DisectLeaf(t *testing.T) {
	type testCase struct {
		in         string
		wantParent string
		wantLeaf   string
		wantErr    error
	}
	tests := []testCase{
		{
			in:         "",
			wantParent: "",
			wantLeaf:   "",
			wantErr:    ErrPathEmpty,
		},
		{
			in:         "/",
			wantParent: "",
			wantLeaf:   "",
			wantErr:    ErrPathEmpty,
		},
		{
			in:         "space1",
			wantParent: "",
			wantLeaf:   "space1",
			wantErr:    nil,
		},
		{
			in:         "/space1/",
			wantParent: "",
			wantLeaf:   "space1",
			wantErr:    nil,
		},
		{
			in:         "space1/space2",
			wantParent: "space1",
			wantLeaf:   "space2",
			wantErr:    nil,
		},
		{
			in:         "/space1/space2/",
			wantParent: "space1",
			wantLeaf:   "space2",
			wantErr:    nil,
		},
		{
			in:         "space1/space2/space3",
			wantParent: "space1/space2",
			wantLeaf:   "space3",
			wantErr:    nil,
		},
		{
			in:         "/space1/space2/space3/",
			wantParent: "space1/space2",
			wantLeaf:   "space3",
			wantErr:    nil,
		},
	}
	for _, tt := range tests {
		gotParent, gotLeaf, gotErr := DisectLeaf(tt.in)
		assert.Equal(t, tt.wantParent, gotParent, "parent isn't matching for %q", tt.in)
		assert.Equal(t, tt.wantLeaf, gotLeaf, "leaf isn't matching for %q", tt.in)
		assert.Equal(t, tt.wantErr, gotErr, "error isn't matching for %q", tt.in)
	}
}

func Test_DisectRoot(t *testing.T) {
	type testCase struct {
		in          string
		wantRoot    string
		wantSubPath string
		wantErr     error
	}
	tests := []testCase{
		{
			in:          "",
			wantRoot:    "",
			wantSubPath: "",
			wantErr:     ErrPathEmpty,
		},
		{
			in:          "/",
			wantRoot:    "",
			wantSubPath: "",
			wantErr:     ErrPathEmpty,
		},
		{
			in:          "space1",
			wantRoot:    "space1",
			wantSubPath: "",
			wantErr:     nil,
		},
		{
			in:          "/space1/",
			wantRoot:    "space1",
			wantSubPath: "",
			wantErr:     nil,
		},
		{
			in:          "space1/space2",
			wantRoot:    "space1",
			wantSubPath: "space2",
			wantErr:     nil,
		},
		{
			in:          "/space1/space2/",
			wantRoot:    "space1",
			wantSubPath: "space2",
			wantErr:     nil,
		},
		{
			in:          "space1/space2/space3",
			wantRoot:    "space1",
			wantSubPath: "space2/space3",
			wantErr:     nil,
		},
		{
			in:          "/space1/space2/space3/",
			wantRoot:    "space1",
			wantSubPath: "space2/space3",
			wantErr:     nil,
		},
	}
	for _, tt := range tests {
		gotRoot, gotSubPath, gotErr := DisectRoot(tt.in)
		assert.Equal(t, tt.wantRoot, gotRoot, "root isn't matching for %q", tt.in)
		assert.Equal(t, tt.wantSubPath, gotSubPath, "subPath isn't matching for %q", tt.in)
		assert.Equal(t, tt.wantErr, gotErr, "error isn't matching for %q", tt.in)
	}
}

func Test_Segments(t *testing.T) {
	type testCase struct {
		in   string
		want []string
	}
	tests := []testCase{
		{
			in:   "",
			want: []string{""},
		},
		{
			in:   "/",
			want: []string{""},
		},
		{
			in:   "space1",
			want: []string{"space1"},
		},
		{
			in:   "/space1/",
			want: []string{"space1"},
		},
		{
			in:   "space1/space2",
			want: []string{"space1", "space2"},
		},
		{
			in:   "/space1/space2/",
			want: []string{"space1", "space2"},
		},
		{
			in:   "space1/space2/space3",
			want: []string{"space1", "space2", "space3"},
		},
		{
			in:   "/space1/space2/space3/",
			want: []string{"space1", "space2", "space3"},
		},
	}
	for _, tt := range tests {
		got := Segments(tt.in)
		assert.Equal(t, tt.want, got, "segments aren't matching for %q", tt.in)
	}
}

func Test_IsAncesterOf(t *testing.T) {
	type testCase struct {
		path  string
		other string
		want  bool
	}
	tests := []testCase{
		{
			path:  "",
			other: "",
			want:  true,
		},
		{
			path:  "space1",
			other: "space1",
			want:  true,
		},
		{
			path:  "space1",
			other: "space1/space2",
			want:  true,
		},
		{
			path:  "space1",
			other: "space1/space2/space3",
			want:  true,
		},
		{
			path:  "space1/space2",
			other: "space1/space2/space3",
			want:  true,
		},
		{
			path:  "/space1/",
			other: "/space1/space2/",
			want:  true,
		},
		{
			path:  "space1",
			other: "space2",
			want:  false,
		},
		{
			path:  "space1/space2",
			other: "space1",
			want:  false,
		},
		{
			path:  "space1/space2",
			other: "space1/space3",
			want:  false,
		},
		{
			path:  "space1",
			other: "space10",
			want:  false,
		},
		{
			path:  "space1/in",
			other: "space1/inner",
			want:  false,
		},
	}
	for _, tt := range tests {
		got := IsAncesterOf(tt.path, tt.other)
		assert.Equal(t, tt.want, got, "IsAncesterOf(%q, %q) isn't matching", tt.path, tt.other)
	}
}

func Test_Parent(t *testing.T) {
	type testCase struct {
		in   string
		want string
	}
	tests := []testCase{
		{
			in:   "",
			want: "",
		},
		{
			in:   "/",
			want: "",
		},
		{
			in:   "space1",
			want: "",
		},
		{
			in:   "/space1/",
			want: "",
		},
		{
			in:   "space1/space2",
			want: "space1",
		},
		{
			in:   "/space1/space2/",
			want: "space1",
		},
		{
			in:   "space1/space2/space3",
			want: "space1/space2",
		},
		{
			in:   "/space1/space2/space3/",
			want: "space1/space2",
		},
	}
	for _, tt := range tests {
		got := Parent(tt.in)
		assert.Equal(t, tt.want, got, "parent isn't matching for %q", tt.in)
	}
}
