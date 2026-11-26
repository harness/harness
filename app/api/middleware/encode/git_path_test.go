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

package encode

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitPathBefore(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		target       string
		wantPath     string
		wantStatus   int
		wantNextCall bool
	}{
		{"dot git suffix", http.MethodGet, "/space1/space2.git", "/space1%2Fspace2", http.StatusOK, true},
		{"dot git with info refs", http.MethodGet, "/space1/space2.git/info/refs?service=git-upload-pack",
			"/space1%2Fspace2/info/refs", http.StatusOK, true},
		{"no dot git, info refs upload pack", http.MethodGet, "/space1/space2/info/refs?service=git-upload-pack",
			"/space1%2Fspace2/info/refs", http.StatusOK, true},
		{"no dot git, info refs receive pack", http.MethodGet, "/a/b/c/info/refs?service=git-receive-pack",
			"/a%2Fb%2Fc/info/refs", http.StatusOK, true},
		{"info refs with unsupported service", http.MethodGet, "/a/b/info/refs?service=git-evil",
			"", http.StatusBadRequest, false},
		{"upload pack", http.MethodPost, "/a/b/git-upload-pack", "/a%2Fb/git-upload-pack", http.StatusOK, true},
		{"receive pack", http.MethodPost, "/a/b/git-receive-pack", "/a%2Fb/git-receive-pack", http.StatusOK, true},
		{"lfs batch", http.MethodPost, "/a/b/info/lfs/objects/batch", "/a%2Fb/info/lfs/objects/batch",
			http.StatusOK, true},
		{"lfs download", http.MethodGet, "/a/b/info/lfs/objects?oid=abc", "/a%2Fb/info/lfs/objects",
			http.StatusOK, true},
		{"lfs upload", http.MethodPut, "/a/b/info/lfs/objects?oid=abc&size=3", "/a%2Fb/info/lfs/objects",
			http.StatusOK, true},
		{"other call is treated as full repo path", http.MethodGet, "/a/b", "/a%2Fb", http.StatusOK, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				called bool
				got    string
			)
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				called = true
				got = r.URL.Path
			})

			rec := httptest.NewRecorder()
			GitPathBefore(next).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.target, nil))

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantNextCall, called)
			if tt.wantNextCall {
				require.Equal(t, tt.wantPath, got)
			}
		})
	}
}

func TestTerminatedRegexPathBefore(t *testing.T) {
	prefixes := []string{`^/registry/([^/]+)/artifact/`}

	tests := []struct {
		name     string
		path     string
		wantPath string
	}{
		{"plain registry", "/registry/abc/artifact/foo/bar/+/summary", "/registry/abc/artifact/foo%2Fbar/summary"},
		{"no marker", "/registry/abc/artifact/foo/bar/summary", "/registry/abc/artifact/foo/bar/summary"},
		{"no prefix match", "/other/abc/artifact/foo/bar/+/summary", "/other/abc/artifact/foo/bar/+/summary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = r.URL.Path
			})

			rec := httptest.NewRecorder()
			TerminatedRegexPathBefore(prefixes, next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, tt.wantPath, got)
		})
	}
}
