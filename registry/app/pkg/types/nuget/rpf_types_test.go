//  Copyright 2023 Harness, Inc.
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

package nuget

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/harness/gitness/registry/app/pkg"

	"github.com/stretchr/testify/require"
)

func TestRpfNugetArtifactInfo(t *testing.T) {
	info := ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{Image: "demo"},
		Version:      "1.0.0",
		Filename:     "demo.1.0.0.nupkg",
	}
	require.Equal(t, "demo", info.BaseArtifactInfo().Image)
	require.Equal(t, "1.0.0", info.GetVersion())
	require.Equal(t, "demo.1.0.0.nupkg", info.GetFileName())

	exists, value := info.GetImageVersion()
	require.True(t, exists)
	require.Equal(t, "demo:1.0.0", value)

	exists, value = ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: "demo"}}.GetImageVersion()
	require.False(t, exists)
	require.Empty(t, value)
}

func TestRpfGetSearchTerm(t *testing.T) {
	tests := []struct {
		name  string
		query url.Values
		want  string
	}{
		{"search term", url.Values{"searchTerm": {"'newtonsoft'"}}, "newtonsoft"},
		{"unquoted search term", url.Values{"searchTerm": {"json"}}, "json"},
		{
			"filter fallback",
			url.Values{"$filter": {"(((Id ne null) and substringof(' microsoft ',tolower(Id)))"}},
			"microsoft",
		},
		{"filter without quotes", url.Values{"$filter": {"Id ne null"}}, ""},
		{"nothing", url.Values{}, ""},
		{"search term wins over filter", url.Values{"searchTerm": {"a"}, "$filter": {"x('b')"}}, "a"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{URL: &url.URL{RawQuery: tc.query.Encode()}}
			require.Equal(t, tc.want, GetSearchTerm(r))
		})
	}
}
