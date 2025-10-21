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

package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetArguments(t *testing.T) {
	tests := []struct {
		name     string
		osArgs   []string
		expected []string
	}{
		{
			name:     "regular command with args",
			osArgs:   []string{"/path/to/gitness", "server", "start"},
			expected: []string{"server", "start"},
		},
		{
			name:     "command with no args",
			osArgs:   []string{"/path/to/gitness"},
			expected: []string{},
		},
		{
			name:     "command with single arg",
			osArgs:   []string{"/path/to/gitness", "version"},
			expected: []string{"version"},
		},
		{
			name:     "command with multiple args",
			osArgs:   []string{"/path/to/gitness", "repo", "create", "myrepo"},
			expected: []string{"repo", "create", "myrepo"},
		},
		{
			name:     "command with flags",
			osArgs:   []string{"/path/to/gitness", "server", "--port", "8080"},
			expected: []string{"server", "--port", "8080"},
		},
		{
			name:     "command with mixed args and flags",
			osArgs:   []string{"/path/to/gitness", "repo", "create", "--private", "myrepo"},
			expected: []string{"repo", "create", "--private", "myrepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args
			originalArgs := os.Args
			defer func() {
				os.Args = originalArgs
			}()

			// Set test args
			os.Args = tt.osArgs

			// Call GetArguments
			result := GetArguments()

			// Verify result
			assert.Equal(t, tt.expected, result, "arguments should match")
		})
	}
}

func TestGetArguments_PreservesOrder(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Set test args with specific order
	os.Args = []string{"/path/to/gitness", "first", "second", "third"}

	// Call GetArguments
	result := GetArguments()

	// Verify order is preserved
	expected := []string{"first", "second", "third"}
	assert.Equal(t, expected, result, "argument order should be preserved")
}

func TestGetArguments_ReturnsSlice(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Set test args
	os.Args = []string{"/path/to/gitness", "arg1", "arg2"}

	// Call GetArguments
	result := GetArguments()

	// Verify result is a slice
	assert.NotNil(t, result, "result should not be nil")
	assert.IsType(t, []string{}, result, "result should be a string slice")
}

func TestGetArguments_EmptyArgs(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Set test args with only command
	os.Args = []string{"/path/to/gitness"}

	// Call GetArguments
	result := GetArguments()

	// Verify result is empty slice
	assert.NotNil(t, result, "result should not be nil")
	assert.Empty(t, result, "result should be empty")
	assert.Equal(t, 0, len(result), "result length should be 0")
}
