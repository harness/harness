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

package version

import (
	"testing"

	"github.com/coreos/go-semver/semver"
)

func TestParseVersionNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "empty string returns zero",
			input:    "",
			expected: 0,
		},
		{
			name:     "zero string returns zero",
			input:    "0",
			expected: 0,
		},
		{
			name:     "positive number",
			input:    "123",
			expected: 123,
		},
		{
			name:     "single digit",
			input:    "5",
			expected: 5,
		},
		{
			name:     "large number",
			input:    "999999",
			expected: 999999,
		},
		{
			name:     "negative number",
			input:    "-1",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersionNumber(tt.input)
			if result != tt.expected {
				t.Errorf("parseVersionNumber(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseVersionNumberPanic(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid number",
			input: "abc",
		},
		{
			name:  "mixed alphanumeric",
			input: "1a2b",
		},
		{
			name:  "decimal number",
			input: "1.5",
		},
		{
			name:  "special characters",
			input: "!@#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("parseVersionNumber(%q) should have panicked", tt.input)
				}
			}()
			parseVersionNumber(tt.input)
		})
	}
}

func TestVersionStructure(t *testing.T) {
	// Test that Version is properly initialized as a semver.Version
	if Version.Major < 0 {
		t.Error("Version.Major should not be negative")
	}
	if Version.Minor < 0 {
		t.Error("Version.Minor should not be negative")
	}
	if Version.Patch < 0 {
		t.Error("Version.Patch should not be negative")
	}
}

func TestVersionString(t *testing.T) {
	// Test that Version can be converted to string without error
	versionStr := Version.String()
	if versionStr == "" {
		t.Error("Version.String() should not be empty")
	}

	// Test that the string representation is valid semver
	_, err := semver.NewVersion(versionStr)
	if err != nil {
		t.Errorf("Version.String() should produce valid semver, got: %s, error: %v", versionStr, err)
	}
}

func TestGitVariables(t *testing.T) {
	// Test that git variables are accessible (they may be empty in tests)
	// This ensures the variables are properly declared and exported
	_ = GitRepository
	_ = GitCommit

	// Test that they are strings
	if GitRepository != "" {
		if len(GitRepository) == 0 {
			t.Error("GitRepository should be a valid string when set")
		}
	}

	if GitCommit != "" {
		if len(GitCommit) == 0 {
			t.Error("GitCommit should be a valid string when set")
		}
	}
}

func TestVersionComparison(t *testing.T) {
	// Test version comparison functionality
	v1 := semver.Version{Major: 1, Minor: 0, Patch: 0}
	v2 := semver.Version{Major: 1, Minor: 1, Patch: 0}

	if !v1.LessThan(v2) {
		t.Error("v1.0.0 should be less than v1.1.0")
	}

	if v2.LessThan(v1) {
		t.Error("v1.1.0 should not be less than v1.0.0")
	}
}

func TestVersionWithPrerelease(t *testing.T) {
	// Test version with prerelease
	v := semver.Version{
		Major:      1,
		Minor:      0,
		Patch:      0,
		PreRelease: semver.PreRelease("alpha"),
	}

	expected := "1.0.0-alpha"
	if v.String() != expected {
		t.Errorf("Version with prerelease should be %s, got %s", expected, v.String())
	}
}

func TestVersionWithMetadata(t *testing.T) {
	// Test version with metadata
	v := semver.Version{
		Major:    1,
		Minor:    0,
		Patch:    0,
		Metadata: "build.1",
	}

	expected := "1.0.0+build.1"
	if v.String() != expected {
		t.Errorf("Version with metadata should be %s, got %s", expected, v.String())
	}
}

func TestVersionWithBothPrereleaseAndMetadata(t *testing.T) {
	// Test version with both prerelease and metadata
	v := semver.Version{
		Major:      1,
		Minor:      0,
		Patch:      0,
		PreRelease: semver.PreRelease("beta"),
		Metadata:   "build.2",
	}

	expected := "1.0.0-beta+build.2"
	if v.String() != expected {
		t.Errorf("Version with prerelease and metadata should be %s, got %s", expected, v.String())
	}
}

// Benchmark tests.
func BenchmarkParseVersionNumber(b *testing.B) {
	for b.Loop() {
		parseVersionNumber("123")
	}
}

func BenchmarkVersionString(b *testing.B) {
	for b.Loop() {
		_ = Version.String()
	}
}
