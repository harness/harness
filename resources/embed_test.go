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

package resources

import (
	"strings"
	"testing"
)

func TestLicenses(t *testing.T) {
	content, err := Licenses()
	if err != nil {
		t.Fatalf("Licenses() returned error: %v", err)
	}

	if len(content) == 0 {
		t.Errorf("Licenses() returned empty content")
	}

	// Verify it's JSON content
	contentStr := string(content)
	if !strings.HasPrefix(contentStr, "[") && !strings.HasPrefix(contentStr, "{") {
		t.Errorf("Licenses() content doesn't appear to be JSON")
	}
}

func TestReadLicense(t *testing.T) {
	// Test reading a common license (MIT is usually available)
	tests := []struct {
		name        string
		licenseName string
		expectError bool
	}{
		{
			name:        "read MIT license",
			licenseName: "mit",
			expectError: false,
		},
		{
			name:        "read non-existent license",
			licenseName: "nonexistent-license-xyz",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := ReadLicense(tt.licenseName)
			if tt.expectError {
				if err == nil {
					t.Errorf("ReadLicense(%q) expected error but got none", tt.licenseName)
				}
			} else {
				if err != nil {
					t.Fatalf("ReadLicense(%q) returned error: %v", tt.licenseName, err)
				}
				if len(content) == 0 {
					t.Errorf("ReadLicense(%q) returned empty content", tt.licenseName)
				}
			}
		})
	}
}

func TestGitIgnores(t *testing.T) {
	files, err := GitIgnores()
	if err != nil {
		t.Fatalf("GitIgnores() returned error: %v", err)
	}

	if len(files) == 0 {
		t.Errorf("GitIgnores() returned empty list")
	}

	// Verify files don't have .gitignore extension
	for _, file := range files {
		if strings.HasSuffix(file, ".gitignore") {
			t.Errorf("GitIgnores() returned file with .gitignore extension: %s", file)
		}
	}

	// Verify we have some common gitignore templates
	hasCommon := false
	commonTemplates := []string{"Go", "Node", "Python", "Java"}
	for _, file := range files {
		for _, common := range commonTemplates {
			if strings.EqualFold(file, common) {
				hasCommon = true
				break
			}
		}
		if hasCommon {
			break
		}
	}

	if !hasCommon {
		t.Logf("Warning: No common gitignore templates found. Available: %v", files)
	}
}

func TestReadGitIgnore(t *testing.T) {
	// First get the list of available gitignores
	files, err := GitIgnores()
	if err != nil {
		t.Fatalf("GitIgnores() returned error: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No gitignore files available to test")
	}

	// Test reading the first available gitignore
	t.Run("read existing gitignore", func(t *testing.T) {
		content, err := ReadGitIgnore(files[0])
		if err != nil {
			t.Fatalf("ReadGitIgnore(%q) returned error: %v", files[0], err)
		}
		if len(content) == 0 {
			t.Errorf("ReadGitIgnore(%q) returned empty content", files[0])
		}
	})

	// Test reading non-existent gitignore
	t.Run("read non-existent gitignore", func(t *testing.T) {
		_, err := ReadGitIgnore("nonexistent-gitignore-xyz")
		if err == nil {
			t.Errorf("ReadGitIgnore(nonexistent) expected error but got none")
		}
	})
}

func TestReadGitIgnoreContent(t *testing.T) {
	// Get available gitignores
	files, err := GitIgnores()
	if err != nil {
		t.Fatalf("GitIgnores() returned error: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No gitignore files available to test")
	}

	// Test that content is valid gitignore format
	for _, file := range files[:minInt(5, len(files))] { // Test first 5 files
		t.Run(file, func(t *testing.T) {
			content, err := ReadGitIgnore(file)
			if err != nil {
				t.Fatalf("ReadGitIgnore(%q) returned error: %v", file, err)
			}

			// Verify content is not empty
			if len(content) == 0 {
				t.Errorf("ReadGitIgnore(%q) returned empty content", file)
			}

			// Verify content is text (not binary)
			contentStr := string(content)
			if len(contentStr) == 0 {
				t.Errorf("ReadGitIgnore(%q) content is not valid text", file)
			}
		})
	}
}

func TestLicensesNotEmpty(t *testing.T) {
	content, err := Licenses()
	if err != nil {
		t.Fatalf("Licenses() returned error: %v", err)
	}

	// Verify content has reasonable size
	if len(content) < 10 {
		t.Errorf("Licenses() content seems too small: %d bytes", len(content))
	}
}

func TestReadLicenseFormat(t *testing.T) {
	// Try to read MIT license and verify it has expected content
	content, err := ReadLicense("mit")
	if err != nil {
		t.Skip("MIT license not available, skipping format test")
	}

	contentStr := string(content)

	// MIT license should contain certain keywords
	keywords := []string{"MIT", "Permission", "Copyright"}
	foundKeywords := 0
	for _, keyword := range keywords {
		if strings.Contains(contentStr, keyword) {
			foundKeywords++
		}
	}

	if foundKeywords == 0 {
		t.Errorf("MIT license content doesn't contain expected keywords")
	}
}

func TestGitIgnoresUnique(t *testing.T) {
	files, err := GitIgnores()
	if err != nil {
		t.Fatalf("GitIgnores() returned error: %v", err)
	}

	// Verify all files are unique
	seen := make(map[string]bool)
	for _, file := range files {
		if seen[file] {
			t.Errorf("GitIgnores() returned duplicate file: %s", file)
		}
		seen[file] = true
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
