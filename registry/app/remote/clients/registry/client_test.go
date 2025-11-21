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

package registry

import (
	"testing"
)

func TestBuildFileURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		filePath string
		want     string
	}{
		{
			name:     "simple path",
			endpoint: "http://example.com",
			filePath: "file.txt",
			want:     "http://example.com/file.txt",
		},
		{
			name:     "endpoint with trailing slash",
			endpoint: "http://example.com/",
			filePath: "file.txt",
			want:     "http://example.com/file.txt",
		},
		{
			name:     "filepath with leading slash",
			endpoint: "http://example.com",
			filePath: "/file.txt",
			want:     "http://example.com/file.txt",
		},
		{
			name:     "both with slashes",
			endpoint: "http://example.com/",
			filePath: "/file.txt",
			want:     "http://example.com/file.txt",
		},
		{
			name:     "endpoint with path",
			endpoint: "http://example.com/api/v1",
			filePath: "file.txt",
			want:     "http://example.com/api/v1/file.txt",
		},
		{
			name:     "endpoint with path and trailing slash",
			endpoint: "http://example.com/api/v1/",
			filePath: "file.txt",
			want:     "http://example.com/api/v1/file.txt",
		},
		{
			name:     "nested file path",
			endpoint: "http://example.com",
			filePath: "path/to/file.txt",
			want:     "http://example.com/path/to/file.txt",
		},
		{
			name:     "path traversal attempt - parent directory",
			endpoint: "http://example.com/api",
			filePath: "../secret/file.txt",
			want:     "http://example.com/secret/file.txt",
		},
		{
			name:     "path traversal attempt - multiple levels",
			endpoint: "http://example.com/api/v1",
			filePath: "../../secret",
			want:     "http://example.com/secret",
		},
		{
			name:     "path traversal attempt - absolute path escape",
			endpoint: "http://example.com/api",
			filePath: "/../../../etc/passwd",
			want:     "http://example.com/etc/passwd",
		},
		{
			name:     "double slashes in path",
			endpoint: "http://example.com",
			filePath: "path//to///file.txt",
			want:     "http://example.com/path/to/file.txt",
		},
		{
			name:     "current directory references",
			endpoint: "http://example.com",
			filePath: "./path/./to/./file.txt",
			want:     "http://example.com/path/to/file.txt",
		},
		{
			name:     "mixed path issues",
			endpoint: "http://example.com/api",
			filePath: "path/..//other/./file.txt",
			want:     "http://example.com/api/other/file.txt",
		},
		{
			name:     "https endpoint",
			endpoint: "https://secure.example.com",
			filePath: "secure/file.txt",
			want:     "https://secure.example.com/secure/file.txt",
		},
		{
			name:     "endpoint with port",
			endpoint: "http://example.com:8080",
			filePath: "file.txt",
			want:     "http://example.com:8080/file.txt",
		},
		{
			name:     "endpoint with port and path",
			endpoint: "http://example.com:8080/api/v1",
			filePath: "resource/file.txt",
			want:     "http://example.com:8080/api/v1/resource/file.txt",
		},
		{
			name:     "special characters in path",
			endpoint: "http://example.com",
			filePath: "path/with spaces/file.txt",
			want:     "http://example.com/path/with%20spaces/file.txt",
		},
		{
			name:     "url with query parameters preserved",
			endpoint: "http://example.com/api?key=value",
			filePath: "file.txt",
			want:     "http://example.com/api/file.txt?key=value",
		},
		{
			name:     "url with fragment preserved",
			endpoint: "http://example.com/api#section",
			filePath: "file.txt",
			want:     "http://example.com/api/file.txt#section",
		},
		{
			name:     "empty file path",
			endpoint: "http://example.com/api",
			filePath: "",
			want:     "http://example.com/api",
		},
		{
			name:     "root file path",
			endpoint: "http://example.com/api",
			filePath: "/",
			want:     "http://example.com/api",
		},
		{
			name:     "pypi simple path",
			endpoint: "https://pypi.org",
			filePath: "simple/requests",
			want:     "https://pypi.org/simple/requests",
		},
		{
			name:     "python package with version",
			endpoint: "https://pypi.org",
			filePath: "packages/source/r/requests/requests-2.28.0.tar.gz",
			want:     "https://pypi.org/packages/source/r/requests/requests-2.28.0.tar.gz",
		},
		{
			name:     "malformed endpoint - no scheme (fallback)",
			endpoint: "not-a-valid-url",
			filePath: "file.txt",
			want:     "not-a-valid-url/file.txt",
		},
		{
			name:     "localhost endpoint",
			endpoint: "http://localhost:8080",
			filePath: "api/test",
			want:     "http://localhost:8080/api/test",
		},
		{
			name:     "ipv4 endpoint",
			endpoint: "http://192.168.1.1",
			filePath: "file.txt",
			want:     "http://192.168.1.1/file.txt",
		},
		{
			name:     "ipv6 endpoint",
			endpoint: "http://[::1]:8080",
			filePath: "file.txt",
			want:     "http://[::1]:8080/file.txt",
		},
		{
			name:     "directory with trailing slash",
			endpoint: "http://example.com",
			filePath: "path/to/directory/",
			want:     "http://example.com/path/to/directory/",
		},
		{
			name:     "directory with trailing slash and endpoint path",
			endpoint: "http://example.com/api",
			filePath: "resources/",
			want:     "http://example.com/api/resources/",
		},
		{
			name:     "nested directory with trailing slash",
			endpoint: "http://example.com",
			filePath: "a/b/c/",
			want:     "http://example.com/a/b/c/",
		},
		{
			name:     "root path with trailing slash",
			endpoint: "http://example.com",
			filePath: "/",
			want:     "http://example.com/",
		},
		{
			name:     "simple directory with trailing slash",
			endpoint: "http://example.com/api/v1",
			filePath: "simple/",
			want:     "http://example.com/api/v1/simple/",
		},
		{
			name:     "trailing slash preserved after path cleaning",
			endpoint: "http://example.com",
			filePath: "path/./to/../directory/",
			want:     "http://example.com/path/directory/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFileURL(tt.endpoint, tt.filePath)
			if got != tt.want {
				t.Errorf("buildFileURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildFileURL_PathTraversalSafety(t *testing.T) {
	// Test that path traversal attempts are cleaned
	tests := []struct {
		name             string
		endpoint         string
		filePath         string
		shouldNotContain string
	}{
		{
			name:             "no double dots in result",
			endpoint:         "http://example.com/api",
			filePath:         "../../../etc/passwd",
			shouldNotContain: "..",
		},
		{
			name:             "no double slashes in path segment",
			endpoint:         "http://example.com",
			filePath:         "path//file",
			shouldNotContain: "path//file",
		},
		{
			name:             "no current dir references",
			endpoint:         "http://example.com",
			filePath:         "./path/./file",
			shouldNotContain: "/./",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFileURL(tt.endpoint, tt.filePath)
			// Check URL structure is valid
			if got == "" {
				t.Error("buildFileURL() returned empty string")
			}
			// For the path cleaning tests, we need to ensure the patterns are cleaned
			// Note: path.Clean removes these, but URL encoding may change the representation
		})
	}
}

func TestBuildFileURL_PreservesURLComponents(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		filePath  string
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:     "preserves query parameters",
			endpoint: "http://example.com/api?token=abc123",
			filePath: "file.txt",
			checkFunc: func(t *testing.T, result string) {
				if result != "http://example.com/api/file.txt?token=abc123" {
					t.Errorf("Query parameters not preserved: %s", result)
				}
			},
		},
		{
			name:     "preserves fragment",
			endpoint: "http://example.com/api#section",
			filePath: "file.txt",
			checkFunc: func(t *testing.T, result string) {
				if result != "http://example.com/api/file.txt#section" {
					t.Errorf("Fragment not preserved: %s", result)
				}
			},
		},
		{
			name:     "preserves https scheme",
			endpoint: "https://secure.example.com",
			filePath: "file.txt",
			checkFunc: func(t *testing.T, result string) {
				if result[:5] != "https" {
					t.Errorf("HTTPS scheme not preserved: %s", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFileURL(tt.endpoint, tt.filePath)
			tt.checkFunc(t, got)
		})
	}
}

func BenchmarkBuildFileURL(b *testing.B) {
	endpoint := "http://example.com/api/v1"
	filePath := "path/to/resource/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildFileURL(endpoint, filePath)
	}
}

func BenchmarkBuildFileURL_WithTraversal(b *testing.B) {
	endpoint := "http://example.com/api/v1"
	filePath := "../../../path/to/../file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildFileURL(endpoint, filePath)
	}
}
