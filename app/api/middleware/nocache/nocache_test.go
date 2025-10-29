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

package nocache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNoCache(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.Handler
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name: "sets no-cache headers",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("test response"))
			}),
			expectedStatus: http.StatusOK,
			checkHeaders:   true,
		},
		{
			name: "preserves handler status code",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
			expectedStatus: http.StatusNotFound,
			checkHeaders:   true,
		},
		{
			name: "works with empty handler",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Empty handler
			}),
			expectedStatus: http.StatusOK,
			checkHeaders:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			// Wrap handler with NoCache middleware
			middleware := NoCache(tt.handler)
			middleware.ServeHTTP(rec, req)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Check no-cache headers
			if tt.checkHeaders {
				expectedHeaders := map[string]string{
					"Expires":         time.Unix(0, 0).Format(time.RFC1123),
					"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
					"Pragma":          "no-cache",
					"X-Accel-Expires": "0",
				}

				for key, expectedValue := range expectedHeaders {
					actualValue := rec.Header().Get(key)
					if actualValue != expectedValue {
						t.Errorf("header %s: expected %q, got %q", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestNoCachePreservesETag(t *testing.T) {
	// Create a handler that sets an ETag
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"test-etag-123"`)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Wrap with NoCache middleware
	middleware := NoCache(handler)
	middleware.ServeHTTP(rec, req)

	// Verify ETag is preserved
	etag := rec.Header().Get("ETag")
	if etag != `"test-etag-123"` {
		t.Errorf("expected ETag to be preserved, got %q", etag)
	}

	// Verify no-cache headers are still set
	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl != "no-cache, no-store, no-transform, must-revalidate, private, max-age=0" {
		t.Errorf("expected Cache-Control header to be set, got %q", cacheControl)
	}
}

func TestNoCacheWithMultipleRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response"))
	})

	middleware := NoCache(handler)

	// Make multiple requests to ensure middleware is reusable
	for i := range 3 {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i, rec.Code)
		}

		expires := rec.Header().Get("Expires")
		if expires == "" {
			t.Errorf("request %d: Expires header not set", i)
		}
	}
}

func TestNoCacheHeaderValues(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware := NoCache(handler)
	middleware.ServeHTTP(rec, req)

	// Verify epoch format
	expectedEpoch := time.Unix(0, 0).Format(time.RFC1123)
	if rec.Header().Get("Expires") != expectedEpoch {
		t.Errorf("Expires header should be epoch time in RFC1123 format")
	}

	// Verify all required directives in Cache-Control
	cacheControl := rec.Header().Get("Cache-Control")
	requiredDirectives := []string{"no-cache", "no-store", "no-transform", "must-revalidate", "private", "max-age=0"}
	for _, directive := range requiredDirectives {
		if !contains(cacheControl, directive) {
			t.Errorf("Cache-Control missing directive: %s", directive)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
