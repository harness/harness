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
