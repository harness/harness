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
	"testing"
)

func TestPattern_Matches(t *testing.T) {
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
