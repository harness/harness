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

package diff

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/harness/gitness/errors"
)

func TestParseFileHeader(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		includePatch  bool
		expectedFile  *File
		expectedError error
	}{
		{
			name:         "valid file header with no quoted paths",
			input:        "diff --git a/example.txt b/example.txt",
			includePatch: false,
			expectedFile: &File{
				Path:    "example.txt",
				OldPath: "example.txt",
				Type:    FileChange,
			},
			expectedError: nil,
		},
		{
			name:         "valid file header with quoted paths",
			input:        `diff --git "a/example space.txt" "b/example space.txt"`,
			includePatch: false,
			expectedFile: &File{
				Path:    "example space.txt",
				OldPath: "example space.txt",
				Type:    FileChange,
			},
			expectedError: nil,
		},
		{
			name:         "valid file header with a as quoted path",
			input:        `diff --git "a/example space.txt" b/example space.txt`,
			includePatch: false,
			expectedFile: &File{
				Path:    "example space.txt",
				OldPath: "example space.txt",
				Type:    FileChange,
			},
			expectedError: nil,
		},
		{
			name:         "valid file header with b as quoted path",
			input:        `diff --git a/example_space.txt "b/example_space.txt"`,
			includePatch: false,
			expectedFile: &File{
				Path:    "example_space.txt",
				OldPath: "example_space.txt",
				Type:    FileChange,
			},
			expectedError: nil,
		},
		{
			name:          "malformed file header",
			input:         "diff --git a/example.txt example.txt",
			includePatch:  false,
			expectedFile:  nil,
			expectedError: &errors.Error{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &Parser{
				Reader:       bufio.NewReader(strings.NewReader(tt.input)),
				IncludePatch: tt.includePatch,
			}

			parser.buffer = []byte(tt.input)

			file, err := parser.parseFileHeader()

			if err != nil && tt.expectedError != nil && !errors.As(err, &tt.expectedError) {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}

			if !reflect.DeepEqual(file, tt.expectedFile) {
				t.Errorf("expected file: %+v, got: %+v", tt.expectedFile, file)
			}
		})
	}
}
