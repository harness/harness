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

package cache

import (
	"reflect"
	"testing"
)

func TestDeduplicate(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "empty",
			input:    nil,
			expected: nil,
		},
		{
			name:     "one-element",
			input:    []int{1},
			expected: []int{1},
		},
		{
			name:     "one-element-duplicated",
			input:    []int{1, 1},
			expected: []int{1},
		},
		{
			name:     "two-elements",
			input:    []int{2, 1},
			expected: []int{1, 2},
		},
		{
			name:     "three-elements",
			input:    []int{2, 2, 3, 3, 1, 1},
			expected: []int{1, 2, 3},
		},
		{
			name:     "many-elements",
			input:    []int{2, 5, 1, 2, 3, 3, 4, 5, 1, 1},
			expected: []int{1, 2, 3, 4, 5},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input = Deduplicate(test.input)
			if want, got := test.expected, test.input; !reflect.DeepEqual(want, got) {
				t.Errorf("failed - want=%v, got=%v", want, got)
				return
			}
		})
	}
}
