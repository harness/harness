// Copyright 2026 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import "testing"

func TestIsInvalidObjectNameError(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    bool
	}{
		{
			name:    "current capitalization",
			message: "fatal: Not a valid object name main",
			want:    true,
		},
		{
			name:    "lowercase capitalization",
			message: "exit status 128: fatal: not a valid object name main",
			want:    true,
		},
		{
			name:    "different git error",
			message: "fatal: not a tree object",
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isInvalidObjectNameError(test.message); got != test.want {
				t.Fatalf("isInvalidObjectNameError(%q) = %t, want %t", test.message, got, test.want)
			}
		})
	}
}
