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

package render

import "testing"

func Test_pagelen(t *testing.T) {
	tests := []struct {
		size, total, want int
	}{
		{25, 1, 1},
		{25, 24, 1},
		{25, 25, 1},
		{25, 26, 2},
		{25, 49, 2},
		{25, 50, 2},
		{25, 51, 3},
	}

	for _, test := range tests {
		got, want := pagelen(test.size, test.total), test.want
		if got != want {
			t.Errorf("got page length %d, want %d", got, want)
		}
	}
}
