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

package protection

import (
	"errors"
	"testing"
)

func TestPattern_Matches(t *testing.T) {
	const defBranch = "default"
	tests := []struct {
		name    string
		pattern Pattern
		input   string
		want    bool
	}{
		{
			name:    "empty-matches-all",
			pattern: Pattern{Default: false, Include: nil, Exclude: nil},
			input:   "blah",
			want:    true,
		},
		{
			name:    "default-matches-default",
			pattern: Pattern{Default: true, Include: nil, Exclude: nil},
			input:   defBranch,
			want:    true,
		},
		{
			name:    "default-mismatches-non-default",
			pattern: Pattern{Default: true, Include: nil, Exclude: nil},
			input:   "non-" + defBranch,
			want:    false,
		},
		{
			name:    "include-matches",
			pattern: Pattern{Default: false, Include: []string{"test*", "dev*"}, Exclude: nil},
			input:   "test123",
			want:    true,
		},
		{
			name:    "include-mismatches",
			pattern: Pattern{Default: false, Include: []string{"test*", "dev*"}, Exclude: nil},
			input:   "marko42",
			want:    false,
		},
		{
			name:    "exclude-matches",
			pattern: Pattern{Default: false, Include: nil, Exclude: []string{"dev*", "pr*"}},
			input:   "blah",
			want:    true,
		},
		{
			name:    "exclude-mismatches",
			pattern: Pattern{Default: false, Include: nil, Exclude: []string{"dev*", "pr*"}},
			input:   "pr_69",
			want:    false,
		},
		{
			name: "complex:not-excluded",
			pattern: Pattern{
				Include: []string{"test/**/*"},
				Exclude: []string{"test/release/*"}},
			input: "test/dev/1",
			want:  true,
		},
		{
			name: "complex:excluded",
			pattern: Pattern{
				Include: []string{"test/**/*"},
				Exclude: []string{"test/release/*"}},
			input: "test/release/1",
			want:  false,
		},
		{
			name: "complex:default-excluded",
			pattern: Pattern{
				Default: true,
				Exclude: []string{defBranch}},
			input: defBranch,
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.pattern.Matches(test.input, defBranch)
			if test.want != got {
				t.Errorf("want=%t got=%t", test.want, got)
			}
		})
	}
}

func TestPattern_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pattern Pattern
		expect  error
	}{
		{
			name:    "empty",
			pattern: Pattern{Default: false, Include: nil, Exclude: nil},
			expect:  nil,
		},
		{
			name:    "default",
			pattern: Pattern{Default: true, Include: nil, Exclude: nil},
			expect:  nil,
		},
		{
			name:    "empty-include-globstar",
			pattern: Pattern{Default: false, Include: []string{""}, Exclude: nil},
			expect:  ErrPatternEmpty,
		},
		{
			name:    "empty-exclude-globstar",
			pattern: Pattern{Default: false, Include: nil, Exclude: []string{""}},
			expect:  ErrPatternEmpty,
		},
		{
			name:    "bad-include-pattern",
			pattern: Pattern{Default: false, Include: []string{"["}, Exclude: nil},
			expect:  ErrInvalidGlobstarPattern,
		},
		{
			name:    "bad-exclude-pattern",
			pattern: Pattern{Default: false, Include: nil, Exclude: []string{"good", "\\"}},
			expect:  ErrInvalidGlobstarPattern,
		},
		{
			name:    "good-pattern",
			pattern: Pattern{Default: true, Include: []string{"test*", "test/**"}, Exclude: []string{"release*"}},
			expect:  nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.pattern.Validate()
			if test.expect == nil && err == nil {
				return
			}

			if !errors.Is(err, test.expect) {
				t.Errorf("want=%v got=%v", test.expect, err)
			}
		})
	}
}

func TestPattern_patternMatches(t *testing.T) {
	tests := []struct {
		pattern  string
		positive []string
		negative []string
	}{
		{
			pattern:  "abc",
			positive: []string{"abc"},
			negative: []string{"abcd", "/abc"},
		},
		{
			pattern:  "*abc",
			positive: []string{"abc", "test-abc"},
			negative: []string{"marko/abc", "abc-test"},
		},
		{
			pattern:  "abc*",
			positive: []string{"abc", "abc-test"},
			negative: []string{"abc/marko", "test-abc"},
		},
		{
			pattern:  "**/abc",
			positive: []string{"abc", "test/abc", "some/other/test/abc"},
			negative: []string{"test/x-abc", "test/abc-x"},
		},
		{
			pattern:  "abc/**",
			positive: []string{"abc", "abc/test", "abc/some/other/test"},
			negative: []string{"test/abc", "x-abc/test"},
		},
		{
			pattern:  "abc[d-e]f",
			positive: []string{"abcdf", "abcef"},
			negative: []string{"abcf", "abcdef"},
		},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			for _, v := range test.positive {
				if ok := patternMatches(test.pattern, v); !ok {
					t.Errorf("pattern=%s positive=%s, got=%t", test.pattern, v, ok)
				}
			}
			for _, v := range test.negative {
				if ok := patternMatches(test.pattern, v); ok {
					t.Errorf("pattern=%s negative=%s, got=%t", test.pattern, v, ok)
				}
			}
		})
	}
}
