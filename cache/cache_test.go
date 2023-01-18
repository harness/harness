// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
			test.input = deduplicate(test.input)
			if want, got := test.expected, test.input; !reflect.DeepEqual(want, got) {
				t.Errorf("failed - want=%v, got=%v", want, got)
				return
			}
		})
	}
}
