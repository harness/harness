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

package request

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplacePrefix(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		rawPath     string
		oldPrefix   string
		newPrefix   string
		wantPath    string
		wantRawPath string
		wantErr     bool
	}{
		{
			name:        "simple path replacement",
			path:        "/api/v1/repos",
			rawPath:     "",
			oldPrefix:   "/api",
			newPrefix:   "/v2",
			wantPath:    "/v2/v1/repos",
			wantRawPath: "",
			wantErr:     false,
		},
		{
			name:        "empty old prefix",
			path:        "/api/v1/repos",
			rawPath:     "",
			oldPrefix:   "",
			newPrefix:   "/v2",
			wantPath:    "/v2/api/v1/repos",
			wantRawPath: "",
			wantErr:     false,
		},
		{
			name:        "empty new prefix",
			path:        "/api/v1/repos",
			rawPath:     "",
			oldPrefix:   "/api",
			newPrefix:   "",
			wantPath:    "/v1/repos",
			wantRawPath: "",
			wantErr:     false,
		},
		{
			name:        "full path replacement",
			path:        "/api",
			rawPath:     "",
			oldPrefix:   "/api",
			newPrefix:   "/v2",
			wantPath:    "/v2",
			wantRawPath: "",
			wantErr:     false,
		},
		{
			name:        "path with raw path",
			path:        "/api/v1/repos",
			rawPath:     "/api/v1/repos",
			oldPrefix:   "/api",
			newPrefix:   "/v2",
			wantPath:    "%2Fv2%2Fv1%2Frepos",
			wantRawPath: "/v2/v1/repos",
			wantErr:     false,
		},
		{
			name:        "path with encoded characters",
			path:        "/api/v1/repos%20test",
			rawPath:     "/api/v1/repos%20test",
			oldPrefix:   "/api",
			newPrefix:   "/v2",
			wantPath:    "%2Fv2%2Fv1%2Frepos%2520test",
			wantRawPath: "/v2/v1/repos%20test",
			wantErr:     false,
		},
		{
			name:        "prefix not found in path",
			path:        "/v1/repos",
			rawPath:     "",
			oldPrefix:   "/api",
			newPrefix:   "/v2",
			wantPath:    "/v1/repos",
			wantRawPath: "",
			wantErr:     true,
		},
		{
			name:        "prefix not found in raw path",
			path:        "/api/v1/repos",
			rawPath:     "/v1/repos",
			oldPrefix:   "/api",
			newPrefix:   "/v2",
			wantPath:    "/api/v1/repos",
			wantRawPath: "/v1/repos",
			wantErr:     true,
		},
		{
			name:        "prefix longer than path",
			path:        "/api",
			rawPath:     "",
			oldPrefix:   "/api/v1",
			newPrefix:   "/v2",
			wantPath:    "/api",
			wantRawPath: "",
			wantErr:     true,
		},
		{
			name:        "partial prefix match should fail",
			path:        "/application/v1",
			rawPath:     "",
			oldPrefix:   "/api",
			newPrefix:   "/v2",
			wantPath:    "/application/v1",
			wantRawPath: "",
			wantErr:     true,
		},
		{
			name:        "replace with longer prefix",
			path:        "/api/v1",
			rawPath:     "",
			oldPrefix:   "/api",
			newPrefix:   "/v2/api/new",
			wantPath:    "/v2/api/new/v1",
			wantRawPath: "",
			wantErr:     false,
		},
		{
			name:        "replace root path",
			path:        "/",
			rawPath:     "",
			oldPrefix:   "/",
			newPrefix:   "/v2",
			wantPath:    "/v2",
			wantRawPath: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new request with the test path
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com"+tt.path, nil)
			require.NoError(t, err)

			// Set raw path if provided
			if tt.rawPath != "" {
				req.URL.RawPath = tt.rawPath
			}

			// Call ReplacePrefix
			err = ReplacePrefix(req, tt.oldPrefix, tt.newPrefix)

			// Check error expectation
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, req.URL.Path, "Path doesn't match")
			assert.Equal(t, tt.wantRawPath, req.URL.RawPath, "RawPath doesn't match")
		})
	}
}

func TestReplacePrefix_PreservesOtherURLFields(t *testing.T) {
	// Create a request with various URL fields set
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"http://example.com:8080/api/v1/repos?query=test#fragment", nil)
	require.NoError(t, err)

	originalScheme := req.URL.Scheme
	originalHost := req.URL.Host
	originalQuery := req.URL.RawQuery
	originalFragment := req.URL.Fragment

	// Replace the prefix
	err = ReplacePrefix(req, "/api", "/v2")
	require.NoError(t, err)

	// Verify other fields are preserved
	assert.Equal(t, originalScheme, req.URL.Scheme, "Scheme should be preserved")
	assert.Equal(t, originalHost, req.URL.Host, "Host should be preserved")
	assert.Equal(t, originalQuery, req.URL.RawQuery, "Query should be preserved")
	assert.Equal(t, originalFragment, req.URL.Fragment, "Fragment should be preserved")
	assert.Equal(t, "/v2/v1/repos", req.URL.Path, "Path should be updated")
}

func TestReplacePrefix_WithComplexURL(t *testing.T) {
	// Test with a complex URL containing special characters
	rawURL := "http://example.com/api/v1/repos/owner%2Frepo?branch=feature%2Ftest&limit=10"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)

	err = ReplacePrefix(req, "/api", "/v2")
	require.NoError(t, err)

	// The path should be updated (note: when RawPath is set, Path gets escaped)
	assert.Equal(t, "%2Fv2%2Fv1%2Frepos%2Fowner%252Frepo", req.URL.Path)
	// Query parameters should be preserved
	assert.Equal(t, "branch=feature%2Ftest&limit=10", req.URL.RawQuery)

	// Verify we can still parse query parameters
	values, err := url.ParseQuery(req.URL.RawQuery)
	require.NoError(t, err)
	assert.Equal(t, "feature/test", values.Get("branch"))
	assert.Equal(t, "10", values.Get("limit"))
}
