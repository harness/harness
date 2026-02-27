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

package generic

import (
	"strings"
	"testing"
)

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
		errMsg   string
	}{
		// Valid file paths
		{
			name:     "simple filename",
			filePath: "file.txt",
			wantErr:  false,
		},
		{
			name:     "filename with allowed characters",
			filePath: "my-file_name.test@version.txt",
			wantErr:  false,
		},
		{
			name:     "nested path with allowed characters",
			filePath: "folder/subfolder/file.txt",
			wantErr:  false,
		},
		{
			name:     "deep nested path",
			filePath: "a/b/c/d/e/file.txt",
			wantErr:  false,
		},
		{
			name:     "filename with tilde",
			filePath: "file~backup.txt",
			wantErr:  false,
		},
		{
			name:     "filename with at symbol",
			filePath: "file@version.txt",
			wantErr:  false,
		},
		{
			name:     "filename with dash and underscore",
			filePath: "my-file_name.txt",
			wantErr:  false,
		},
		{
			name:     "alphanumeric path",
			filePath: "folder123/file456.txt",
			wantErr:  false,
		},

		// Invalid file paths - regex pattern violations
		{
			name:     "empty string",
			filePath: "",
			wantErr:  true,
		},
		{
			name:     "whitespace only",
			filePath: "   ",
			wantErr:  true,
		},
		{
			name:     "starts with slash",
			filePath: "/file.txt",
			wantErr:  true,
		},
		{
			name:     "contains spaces",
			filePath: "file name.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file name.txt\", should follow pattern:",
		},
		{
			name:     "contains special characters",
			filePath: "file!.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file!.txt\", should follow pattern:",
		},
		{
			name:     "contains hash",
			filePath: "file#.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file#.txt\", should follow pattern:",
		},
		{
			name:     "contains percent",
			filePath: "file%.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file%.txt\", should follow pattern:",
		},
		{
			name:     "contains ampersand",
			filePath: "file&.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file&.txt\", should follow pattern:",
		},
		{
			name:     "contains asterisk",
			filePath: "file*.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file*.txt\", should follow pattern:",
		},
		{
			name:     "contains question mark",
			filePath: "file?.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file?.txt\", should follow pattern:",
		},
		{
			name:     "contains pipe",
			filePath: "file|.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file|.txt\", should follow pattern:",
		},
		{
			name:     "contains backslash",
			filePath: "file\\.txt",
			wantErr:  true,
		},
		{
			name:     "contains colon",
			filePath: "file:.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file:.txt\", should follow pattern:",
		},
		{
			name:     "contains semicolon",
			filePath: "file;.txt",
			wantErr:  true,
		},
		{
			name:     "contains quotes",
			filePath: "file\".txt",
			wantErr:  true,
		},
		{
			name:     "contains single quote",
			filePath: "file'.txt",
			wantErr:  true,
		},
		{
			name:     "contains less than",
			filePath: "file<.txt",
			wantErr:  true,
		},
		{
			name:     "contains greater than",
			filePath: "file>.txt",
			wantErr:  true,
			errMsg:   "invalid file path: \"file>.txt\", should follow pattern:",
		},

		// Invalid file paths - relative path segments
		{
			name:     "current directory reference",
			filePath: "./file.txt",
			wantErr:  true,
			errMsg:   "relative segments not allowed in file path:",
		},
		{
			name:     "parent directory reference",
			filePath: "../file.txt",
			wantErr:  true,
		},
		{
			name:     "current directory in middle",
			filePath: "folder/./file.txt",
			wantErr:  true,
			errMsg:   "relative segments not allowed in file path:",
		},
		{
			name:     "parent directory in middle",
			filePath: "folder/../file.txt",
			wantErr:  true,
			errMsg:   "relative segments not allowed in file path:",
		},
		{
			name:     "multiple relative segments",
			filePath: "folder/../../file.txt",
			wantErr:  true,
			errMsg:   "relative segments not allowed in file path:",
		},

		// Invalid file paths - unsafe path elements
		{
			name:     "empty path segment",
			filePath: "folder//file.txt",
			wantErr:  true,
		},
		{
			name:     "dot segment",
			filePath: "folder/./file.txt",
			wantErr:  true,
			errMsg:   "relative segments not allowed in file path:", // This will be caught by path.Clean first
		},
		{
			name:     "double dot segment",
			filePath: "folder/../file.txt",
			wantErr:  true,
			errMsg:   "relative segments not allowed in file path:", // This will be caught by path.Clean first
		},

		// Invalid file paths - length validation
		{
			name:     "path too long",
			filePath: strings.Repeat("a", maxFilePathLength+1),
			wantErr:  true,
			errMsg:   "file path too long:",
		},
		{
			name:     "path at max length",
			filePath: strings.Repeat("a", maxFilePathLength),
			wantErr:  false,
		},

		// Edge cases
		{
			name:     "single character",
			filePath: "a",
			wantErr:  false,
		},
		{
			name:     "single dot",
			filePath: ".",
			wantErr:  true,
			errMsg:   "unsafe path element",
		},
		{
			name:     "double dots",
			filePath: "..",
			wantErr:  true,
			errMsg:   "unsafe path element",
		},
		{
			name:     "trailing slash",
			filePath: "folder/",
			wantErr:  true,
		},
		{
			name:     "multiple slashes",
			filePath: "folder///file.txt",
			wantErr:  true,
		},

		// Valid edge cases with whitespace
		{
			name:     "path with leading whitespace gets trimmed",
			filePath: "  file.txt",
			wantErr:  true,
		},
		{
			name:     "path with trailing whitespace gets trimmed",
			filePath: "file.txt  ",
			wantErr:  true,
		},
		{
			name:     "path with leading and trailing whitespace gets trimmed",
			filePath: "  file.txt  ",
			wantErr:  true,
		},

		// Additional regex pattern tests
		{
			name:     "starts with number",
			filePath: "123file.txt",
			wantErr:  false,
		},
		{
			name:     "starts with letter",
			filePath: "afile.txt",
			wantErr:  false,
		},
		{
			name:     "ends with number",
			filePath: "file123",
			wantErr:  false,
		},
		{
			name:     "ends with letter",
			filePath: "filea",
			wantErr:  false,
		},
		{
			name:     "mixed case",
			filePath: "MyFile.TXT",
			wantErr:  false,
		},
		{
			name:     "exactly max length",
			filePath: strings.Repeat("a", maxFilePathLength),
			wantErr:  false,
		},
		{
			name:     "one character over max length",
			filePath: strings.Repeat("a", maxFilePathLength+1),
			wantErr:  true,
		},
		{
			name:     "long period",
			filePath: ".........abc",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.filePath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateFilePath() expected error but got nil for input: %q", tt.filePath)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateFilePath() error = %v, expected to contain %q", err, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("validateFilePath() unexpected error = %v for input: %q", err, tt.filePath)
			}
		})
	}
}

// TestValidateFilePathRegexPattern tests the regex pattern specifically.
func TestValidateFilePathRegexPattern(t *testing.T) {
	validPaths := []string{
		"file.txt",
		"folder/file.txt",
		"a/b/c/file.txt",
		"my-file_name.test@version~backup.txt",
		"123",
		"a",
		"file123",
		"MyFile.TXT",
	}

	invalidPaths := []string{
		"file name.txt", // space
		"file!.txt",     // exclamation
		"file#.txt",     // hash
		"file$.txt",     // dollar
		"file%.txt",     // percent
		"file&.txt",     // ampersand
		"file*.txt",     // asterisk
		"file+.txt",     // plus
		"file=.txt",     // equals
		"file?.txt",     // question mark
		"file^.txt",     // caret
		"file|.txt",     // pipe
		"file\\.txt",    // backslash
		"file:.txt",     // colon
		"file;.txt",     // semicolon
		"file\".txt",    // quote
		"file'.txt",     // single quote
		"file<.txt",     // less than
		"file>.txt",     // greater than
		"file[.txt",     // left bracket
		"file].txt",     // right bracket
		"file{.txt",     // left brace
		"file}.txt",     // right brace
		"file(.txt",     // left paren
		"file).txt",     // right paren
		"file,.txt",     // comma
	}

	for _, path := range validPaths {
		t.Run("valid_regex_"+path, func(t *testing.T) {
			if !filePathRe.MatchString(path) {
				t.Errorf("filePathRe should match valid path: %q", path)
			}
		})
	}

	for _, path := range invalidPaths {
		t.Run("invalid_regex_"+path, func(t *testing.T) {
			if filePathRe.MatchString(path) {
				t.Errorf("filePathRe should not match invalid path: %q", path)
			}
		})
	}
}

// Test helpers for URL path parsing

type urlPathTest struct {
	name         string
	urlPath      string
	wantSegments []string
	wantPackage  string
	wantVersion  string
	wantFileName string
	wantFilePath string
	isGeneric    bool
}

func validateGenericPackageSegments(t *testing.T, segments []string, tt urlPathTest) {
	t.Helper()
	if len(segments) < 3 {
		t.Errorf("GENERIC package needs at least 3 segments")
		return
	}

	packageName := segments[0]
	version := segments[1]
	fileName := segments[len(segments)-1]
	filePath := strings.Join(segments[2:], "/")

	if packageName != tt.wantPackage {
		t.Errorf("Expected package=%s, got %s", tt.wantPackage, packageName)
	}
	if version != tt.wantVersion {
		t.Errorf("Expected version=%s, got %s", tt.wantVersion, version)
	}
	if fileName != tt.wantFileName {
		t.Errorf("Expected fileName=%s, got %s", tt.wantFileName, fileName)
	}
	if filePath != tt.wantFilePath {
		t.Errorf("Expected filePath=%s, got %s", tt.wantFilePath, filePath)
	}
}

func validateNonGenericPackageSegments(t *testing.T, segments []string, tt urlPathTest) {
	t.Helper()
	fileName := segments[len(segments)-1]
	filePath := strings.Join(segments, "/")

	if fileName != tt.wantFileName {
		t.Errorf("Expected fileName=%s, got %s", tt.wantFileName, fileName)
	}
	if filePath != tt.wantFilePath {
		t.Errorf("Expected filePath=%s, got %s", tt.wantFilePath, filePath)
	}
}

func TestURLPathParsing(t *testing.T) {
	tests := []urlPathTest{
		{
			name:         "GENERIC package - simple",
			urlPath:      "/pkg/root/registry/files/mypackage/1.0.0/file.jar",
			wantSegments: []string{"mypackage", "1.0.0", "file.jar"},
			wantPackage:  "mypackage",
			wantVersion:  "1.0.0",
			wantFileName: "file.jar",
			wantFilePath: "file.jar",
			isGeneric:    true,
		},
		{
			name:         "GENERIC package - nested path",
			urlPath:      "/pkg/root/registry/files/org.example/2.0.0/sub/dir/file.jar",
			wantSegments: []string{"org.example", "2.0.0", "sub", "dir", "file.jar"},
			wantPackage:  "org.example",
			wantVersion:  "2.0.0",
			wantFileName: "file.jar",
			wantFilePath: "sub/dir/file.jar",
			isGeneric:    true,
		},
		{
			name:         "non-GENERIC package - MAVEN",
			urlPath:      "/pkg/root/registry/files/org/example/lib/1.0/file.jar",
			wantSegments: []string{"org", "example", "lib", "1.0", "file.jar"},
			wantFileName: "file.jar",
			wantFilePath: "org/example/lib/1.0/file.jar",
			isGeneric:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := strings.TrimPrefix(tt.urlPath, "/")
			splits := strings.Split(path, "/")
			// splits[0] = "pkg", splits[1] = root, splits[2] = registry, splits[3] = "files"
			remainingSegments := splits[4:]

			if len(remainingSegments) != len(tt.wantSegments) {
				t.Errorf("Expected %d segments, got %d", len(tt.wantSegments), len(remainingSegments))
			}

			if tt.isGeneric {
				validateGenericPackageSegments(t, remainingSegments, tt)
			} else {
				validateNonGenericPackageSegments(t, remainingSegments, tt)
			}
		})
	}
}
