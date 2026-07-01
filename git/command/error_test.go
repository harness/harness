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

package command

import (
	"errors"
	"testing"
)

func TestError_IsSHAMismatchErr(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name: "SHA mismatch error",
			errMsg: "fatal: cannot lock ref 'refs/pullreq/123/head': " +
				"is at 6e1f706a11a21cd69fd4c56426a09ace4ac0ed71 but expected 7c4657be9a67452b897e07484d6a1099a7e5e07f",
			expected: true,
		},
		{
			name:     "reference already exists",
			errMsg:   "fatal: cannot lock ref 'refs/heads/test': reference already exists",
			expected: false,
		},
		{
			name:     "unable to resolve reference",
			errMsg:   "fatal: cannot lock ref 'refs/heads/nonexistent': unable to resolve reference 'refs/heads/nonexistent'",
			expected: false,
		},
		{
			name:     "not a valid ref",
			errMsg:   "fatal: 'bad..ref' is not a valid ref",
			expected: false,
		},
		{
			name:     "generic error",
			errMsg:   "some other git error",
			expected: false,
		},
		{
			name:     "partial match - missing 'cannot lock ref'",
			errMsg:   "is at abc but expected def",
			expected: false,
		},
		{
			name:     "partial match - missing 'is at'",
			errMsg:   "fatal: cannot lock ref 'refs/heads/test': but expected something",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(errors.New(tt.errMsg), []byte(tt.errMsg))
			result := err.IsSHAMismatchErr()
			if result != tt.expected {
				t.Errorf("IsSHAMismatchErr() = %v, expected %v for error: %s", result, tt.expected, tt.errMsg)
			}
		})
	}
}
