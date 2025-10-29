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

package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: true,
		},
		{
			name:     "empty string slice",
			input:    []string{},
			expected: true,
		},
		{
			name:     "non-empty string slice",
			input:    []string{"a", "b"},
			expected: false,
		},
		{
			name:     "empty int slice",
			input:    []int{},
			expected: true,
		},
		{
			name:     "non-empty int slice",
			input:    []int{1, 2, 3},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmpty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJoinWithSeparator(t *testing.T) {
	tests := []struct {
		name     string
		sep      string
		args     []string
		expected string
	}{
		{
			name:     "join with comma",
			sep:      ",",
			args:     []string{"a", "b", "c"},
			expected: "a,b,c",
		},
		{
			name:     "join with space",
			sep:      " ",
			args:     []string{"hello", "world"},
			expected: "hello world",
		},
		{
			name:     "empty strings",
			sep:      "-",
			args:     []string{"", "", ""},
			expected: "--",
		},
		{
			name:     "single string",
			sep:      ".",
			args:     []string{"single"},
			expected: "single",
		},
		{
			name:     "no strings",
			sep:      "+",
			args:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinWithSeparator(tt.sep, tt.args...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFirstNumber(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:        "simple number",
			input:       "123",
			expected:    123,
			expectError: false,
		},
		{
			name:        "number with text",
			input:       "abc123def",
			expected:    123,
			expectError: false,
		},
		{
			name:        "multiple numbers",
			input:       "123abc456",
			expected:    123,
			expectError: false,
		},
		{
			name:        "no numbers",
			input:       "abc",
			expected:    0,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractFirstNumber(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
