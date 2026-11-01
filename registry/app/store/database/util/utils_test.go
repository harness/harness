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

package util

import (
	"testing"
)

func TestSanitizeSortOrder(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "ASC",
		},
		{
			name:     "whitespace",
			input:    "   ",
			expected: "ASC",
		},
		{
			name:     "uppercase_asc",
			input:    "ASC",
			expected: "ASC",
		},
		{
			name:     "lowercase_asc",
			input:    "asc",
			expected: "ASC",
		},
		{
			name:     "uppercase_desc",
			input:    "DESC",
			expected: "DESC",
		},
		{
			name:     "lowercase_desc",
			input:    "desc",
			expected: "DESC",
		},
		{
			name:     "mixed_case_desc",
			input:    "DeSc",
			expected: "DESC",
		},
		{
			name:     "desc_with_whitespace",
			input:    "  DESC  ",
			expected: "DESC",
		},
		{
			name:     "arbitrary_string",
			input:    "invalid",
			expected: "ASC",
		},
		{
			name:     "sqli_attempt_order_by",
			input:    "ASC,(SELECT 1)",
			expected: "ASC",
		},
		{
			name:     "sqli_attempt_desc_order_by",
			input:    "DESC,(SELECT 1)",
			expected: "ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeSortOrder(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeSortOrder(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}
