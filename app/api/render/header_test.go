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

package render

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPaginationLinkQuery(t *testing.T) {
	r := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/repos?access_token=secret&token=secret&query=abc",
		nil,
	)
	w := httptest.NewRecorder()

	Pagination(r, w, 2, 10, 30)

	links := w.Header().Values("Link")
	if got, want := len(links), 3; got != want {
		t.Fatalf("Want %d link headers, got %d: %v", want, got, links)
	}

	for _, link := range links {
		rawURL, _, _ := strings.Cut(strings.TrimPrefix(link, "<"), ">")
		u, err := url.Parse(rawURL)
		if err != nil {
			t.Fatalf("Failed to parse link %q: %s", link, err)
		}

		params := u.Query()
		if params.Has("access_token") || params.Has("token") {
			t.Errorf("Want link %q without access tokens", link)
		}
		if got, want := params.Get("limit"), "10"; got != want {
			t.Errorf("Want limit %q in link %q, got %q", want, link, got)
		}
		if got, want := params.Get("query"), "abc"; got != want {
			t.Errorf("Want query %q in link %q, got %q", want, link, got)
		}
	}
}
