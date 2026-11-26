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

package lib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRrmLinkString(t *testing.T) {
	lk := &Link{
		URL: "http://example.com/page",
		Rel: "next",
		Attrs: map[string]string{
			"title": "next page",
		},
	}
	got := lk.String()
	require.Contains(t, got, `<http://example.com/page>`)
	require.Contains(t, got, `rel="next"`)
	require.Contains(t, got, `title="next page"`)
}

func TestRrmLinksString(t *testing.T) {
	links := Links{
		{URL: "http://a.example", Rel: "prev"},
		{URL: "http://b.example", Rel: "next"},
	}
	got := links.String()
	require.Contains(t, got, "http://a.example")
	require.Contains(t, got, "http://b.example")
}

func TestRrmParseLinks(t *testing.T) {
	raw := `<http://example.com/TheBook/chapter2>; rel="previous"; title="prev" , ` +
		`<http://example.com/TheBook/chapter4>; rel="next"; title="next"`
	links := ParseLinks(raw)
	require.Len(t, links, 2)
	require.Equal(t, "http://example.com/TheBook/chapter2", links[0].URL)
	require.Equal(t, "previous", links[0].Rel)
	require.Equal(t, "prev", links[0].Attrs["title"])
	require.Equal(t, "next", links[1].Rel)
}

func TestRrmParseLinksSkipsMissingURL(t *testing.T) {
	links := ParseLinks(`; rel="next"`)
	require.Empty(t, links)
}
