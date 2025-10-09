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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerminatedPathBefore(t *testing.T) {
	tests := []struct {
		name           string
		prefixes       []string
		path           string
		expectedPath   string
		expectedStatus int
	}{
		{
			name:           "basic terminated path",
			prefixes:       []string{"/spaces"},
			path:           "/spaces/space1/space2/+/authToken",
			expectedPath:   "/spaces/space1%2Fspace2/authToken",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no matching prefix",
			prefixes:       []string{"/other"},
			path:           "/spaces/space1/space2/+/authToken",
			expectedPath:   "/spaces/space1/space2/+/authToken",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "multiple prefixes - first match",
			prefixes:       []string{"/spaces", "/repos"},
			path:           "/spaces/space1/space2/+/authToken",
			expectedPath:   "/spaces/space1%2Fspace2/authToken",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "encoded paths - no match",
			prefixes:       []string{"/spaces", "/repos"},
			path:           "/spaces/space1%2Fspace2/authToken",
			expectedPath:   "/spaces/space1%2Fspace2/authToken",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "encoded paths - suffix match 1",
			prefixes:       []string{"/spaces", "/repos"},
			path:           "/spaces/space1/space2/+/",
			expectedPath:   "/spaces/space1%2Fspace2/",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "encoded paths - suffix match 2",
			prefixes:       []string{"/spaces", "/repos"},
			path:           "/spaces/space1/space2/+",
			expectedPath:   "/spaces/space1%2Fspace2",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path:    tt.path,
					RawPath: tt.path,
				},
			}
			rr := httptest.NewRecorder()

			handler := TerminatedPathBefore(tt.prefixes, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.expectedPath, r.URL.RawPath)
			}))

			handler.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestCutOutTerminatedPath(t *testing.T) {
	tests := []struct {
		name          string
		subPath       string
		marker        string
		expectedPath  string
		expectedFound bool
	}{
		{
			name:          "basic path with marker",
			subPath:       "/space1/space2/+/authToken",
			marker:        "/+",
			expectedPath:  "/space1/space2",
			expectedFound: true,
		},
		{
			name:          "git suffix",
			subPath:       "/space1/space2.git",
			marker:        ".git",
			expectedPath:  "/space1/space2",
			expectedFound: true,
		},
		{
			name:          "git suffix with trailing slash",
			subPath:       "/space1/space2.git/",
			marker:        ".git",
			expectedPath:  "/space1/space2",
			expectedFound: true,
		},
		{
			name:          "multiple markers 1",
			subPath:       "/foo/bar/+/artifact/myartifact/+",
			marker:        "/+",
			expectedPath:  "/foo/bar",
			expectedFound: true,
		},
		{
			name:          "multiple markers 2",
			subPath:       "/foo/bar/+/artifact/myartifact/+/",
			marker:        "/+",
			expectedPath:  "/foo/bar",
			expectedFound: true,
		},
		{
			name:          "mixed markers 1",
			subPath:       "/foo/bar/+artifact/myartifact/+",
			marker:        "/+",
			expectedPath:  "/foo/bar/+artifact/myartifact",
			expectedFound: true,
		},
		{
			name:          "mixed markers 2",
			subPath:       "/foo/bar/repo.git/myfile1/myfile2/.gitignore",
			marker:        ".git",
			expectedPath:  "/foo/bar/repo",
			expectedFound: true,
		},
		{
			name:          "mixed markers 3",
			subPath:       "/foo/bar/+artifact/myartifact",
			marker:        "/+",
			expectedPath:  "",
			expectedFound: false,
		},
		{
			name:          "no marker found",
			subPath:       "/space1/space2",
			marker:        "/+",
			expectedPath:  "",
			expectedFound: false,
		},

		{
			name:          "path with non-slash marker as suffix",
			subPath:       "/space1/space2/repo.git",
			marker:        ".git",
			expectedPath:  "/space1/space2/repo",
			expectedFound: true,
		},
		{
			name:          "path with non-slash marker as suffix with slash",
			subPath:       "/space1/space2/repo.git/",
			marker:        ".git",
			expectedPath:  "/space1/space2/repo",
			expectedFound: true,
		},
		{
			name:          "path with non-slash marker in the middle",
			subPath:       "/space1/space2/repo.git/authToken",
			marker:        ".git",
			expectedPath:  "/space1/space2/repo",
			expectedFound: true,
		},
		{
			name:          "path with multiple non-slash markers",
			subPath:       "/space1/space2/repo.git/authToken.git",
			marker:        ".git",
			expectedPath:  "/space1/space2/repo",
			expectedFound: true,
		},
		{
			name:          "path with slash marker as suffix",
			subPath:       "/space1/space2/+",
			marker:        "/+",
			expectedPath:  "/space1/space2",
			expectedFound: true,
		},
		{
			name:          "path with slash marker as suffix with slash",
			subPath:       "/space1/space2/+/",
			marker:        "/+",
			expectedPath:  "/space1/space2",
			expectedFound: true,
		},
		{
			name:          "path with slash marker as suffix with without slash",
			subPath:       "/space1/space2/+",
			marker:        "/+/",
			expectedPath:  "",
			expectedFound: false,
		},
		{
			name:          "path with slash marker in the middle",
			subPath:       "/space1/space2/+/authToken",
			marker:        "/+",
			expectedPath:  "/space1/space2",
			expectedFound: true,
		},
		{
			name:          "path with slash marker in the middle without trailing slash",
			subPath:       "/space1/space2/+authToken",
			marker:        "/+",
			expectedPath:  "",
			expectedFound: false,
		},
		{
			name:          "encoded path with no marker",
			subPath:       "/spaces/space1%2Fspace2/authToken",
			marker:        "/+",
			expectedPath:  "",
			expectedFound: false,
		},
		{
			name:          "path with exact length 1",
			subPath:       "/+/",
			marker:        "/+",
			expectedPath:  "",
			expectedFound: true,
		},
		{
			name:          "path with exact length 2",
			subPath:       "/+",
			marker:        "/+",
			expectedPath:  "",
			expectedFound: true,
		},
		{
			name:          "empty marker",
			subPath:       "/foo/bar",
			marker:        "",
			expectedPath:  "/foo/bar",
			expectedFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, found := cutOutTerminatedPath(tt.subPath, tt.marker)
			assert.Equal(t, tt.expectedPath, path)
			assert.Equal(t, tt.expectedFound, found)
		})
	}
}

func TestRegexPathTerminatedWithMarker(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		regexPrefix    string
		marker         string
		expectedPath   string
		expectedStatus int
	}{
		{
			name:           "registry artifact path",
			path:           "/registry/app1%2Fremote2/artifact/foo/bar/+/summary",
			regexPrefix:    "^/registry/([^/]+)/artifact/",
			marker:         "/+",
			expectedPath:   "/registry/app1%2Fremote2/artifact/foo%2Fbar/summary",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no regex match",
			path:           "/other/path/foo/bar/+/summary",
			regexPrefix:    "^/registry/([^/]+)/artifact/",
			marker:         "/+",
			expectedPath:   "/other/path/foo/bar/+/summary",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path:    tt.path,
					RawPath: tt.path,
				},
			}
			rr := httptest.NewRecorder()

			handler := TerminatedRegexPathBefore([]string{tt.regexPrefix},
				http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					assert.Equal(t, tt.expectedPath, r.URL.RawPath)
				}))

			handler.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
