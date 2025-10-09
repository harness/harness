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

package enum

import "testing"

func TestParseOrder(t *testing.T) {
	tests := []struct {
		text string
		want Order
	}{
		{"asc", OrderAsc},
		{"Asc", OrderAsc},
		{"ASC", OrderAsc},
		{"ascending", OrderAsc},
		{"Ascending", OrderAsc},
		{"desc", OrderDesc},
		{"Desc", OrderDesc},
		{"DESC", OrderDesc},
		{"descending", OrderDesc},
		{"Descending", OrderDesc},
		{"", OrderDefault},
		{"invalid", OrderDefault},
	}

	for _, test := range tests {
		got, want := ParseOrder(test.text), test.want
		if got != want {
			t.Errorf("Want order %q parsed as %q, got %q", test.text, want, got)
		}
	}
}
