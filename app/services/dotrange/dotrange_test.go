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

package dotrange

import (
	"testing"

	"github.com/harness/gitness/app/api/usererror"

	"github.com/stretchr/testify/require"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want DotRange
	}{
		{"two dots", "main..feature", DotRange{BaseRef: "main", HeadRef: "feature"}},
		{"three dots use merge base", "main...feature", DotRange{BaseRef: "main", HeadRef: "feature", MergeBase: true}},
		{"upstream head", "main..upstream:feature", DotRange{BaseRef: "main", HeadRef: "feature", HeadUpstream: true}},
		{"upstream base", "upstream:main...feature",
			DotRange{BaseRef: "main", BaseUpstream: true, HeadRef: "feature", MergeBase: true}},
		{"refs with slashes", "release/1.0..feature/x", DotRange{BaseRef: "release/1.0", HeadRef: "feature/x"}},
		{"empty sides are kept", "..", DotRange{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePath(tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	t.Run("missing separator", func(t *testing.T) {
		_, err := ParsePath("main")
		require.Error(t, err)
		var userErr *usererror.Error
		require.ErrorAs(t, err, &userErr)
		require.Contains(t, userErr.Message, "Invalid format")
	})

	t.Run("two upstream references", func(t *testing.T) {
		_, err := ParsePath("upstream:main..upstream:feature")
		require.Error(t, err)
		var userErr *usererror.Error
		require.ErrorAs(t, err, &userErr)
		require.Contains(t, userErr.Message, "Only one upstream reference")
	})
}

func TestMake(t *testing.T) {
	got, err := Make("a", "b", true)
	require.NoError(t, err)
	require.Equal(t, DotRange{BaseRef: "a", HeadRef: "b", MergeBase: true}, got)

	got, err = Make("upstream:a", "b", false)
	require.NoError(t, err)
	require.Equal(t, DotRange{BaseRef: "a", BaseUpstream: true, HeadRef: "b"}, got)

	_, err = Make("upstream:a", "upstream:b", false)
	require.Error(t, err)
}

func TestDotRange_String(t *testing.T) {
	tests := []struct {
		r    DotRange
		want string
	}{
		{DotRange{BaseRef: "main", HeadRef: "feature"}, "main..feature"},
		{DotRange{BaseRef: "main", HeadRef: "feature", MergeBase: true}, "main...feature"},
		{DotRange{BaseRef: "main", BaseUpstream: true, HeadRef: "feature"}, "upstream:main..feature"},
		{DotRange{BaseRef: "main", HeadRef: "feature", HeadUpstream: true, MergeBase: true}, "main...upstream:feature"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, tt.r.String())
	}
}

func TestParsePathRoundTrip(t *testing.T) {
	for _, path := range []string{"a..b", "a...b", "upstream:a..b", "a...upstream:b"} {
		parsed, err := ParsePath(path)
		require.NoError(t, err)
		require.Equal(t, path, parsed.String())
	}
}
