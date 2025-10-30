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
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestParseTagsFromQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected map[string][]string
	}{
		{
			name:     "no tags",
			query:    "",
			expected: nil,
		},
		{
			name:  "single key-only",
			query: "tag=backend",
			expected: map[string][]string{
				"backend": nil,
			},
		},
		{
			name:  "single key-value",
			query: "tag=backend:golang",
			expected: map[string][]string{
				"backend": {"golang"},
			},
		},
		{
			name:  "multiple values same key",
			query: "tag=backend:golang&tag=backend:java",
			expected: map[string][]string{
				"backend": {"golang", "java"},
			},
		},
		{
			name:  "duplicate values deduped",
			query: "tag=backend:golang&tag=backend:golang",
			expected: map[string][]string{
				"backend": {"golang"},
			},
		},
		{
			name:  "key-only dominates",
			query: "tag=backend&tag=backend:golang&tag=backend:java",
			expected: map[string][]string{
				"backend": nil,
			},
		},
		{
			name:  "multiple different keys",
			query: "tag=backend:golang&tag=db:postgres",
			expected: map[string][]string{
				"backend": {"golang"},
				"db":      {"postgres"},
			},
		},
		{
			name:  "multiple different keys with duplicates",
			query: "tag=backend:python&tag=backend:golang&tag=db:postgres&tag=backend:golang&tag=db:postgres",
			expected: map[string][]string{
				"backend": {"golang", "python"},
				"db":      {"postgres"},
			},
		},
		{
			name:  "empty value",
			query: "tag=backend:",
			expected: map[string][]string{
				"backend": {""},
			},
		},
		{
			name:  "mixed key-only and value keys",
			query: "tag=backend&tag=db:postgres&tag=auth:jwt",
			expected: map[string][]string{
				"backend": nil,
				"db":      {"postgres"},
				"auth":    {"jwt"},
			},
		},
		{
			name:  "empty key is preserved",
			query: "tag=:value&tag=:",
			expected: map[string][]string{
				"": {"", "value"},
			},
		},
		{
			name:  "empty key-only",
			query: "tag=",
			expected: map[string][]string{
				"": nil,
			},
		},
		{
			name:  "two empty key-only",
			query: "tag=&tag=",
			expected: map[string][]string{
				"": nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse("http://example.com/?" + tt.query)
			req := &http.Request{URL: u}

			got := ParseTagsFromQuery(req)

			// compare maps
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("expected %#v, got %#v", tt.expected, got)
			}
		})
	}
}
