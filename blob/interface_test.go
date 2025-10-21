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

package blob

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrNotFound",
			err:      ErrNotFound,
			expected: "resource not found",
		},
		{
			name:     "ErrNotSupported",
			err:      ErrNotSupported,
			expected: "not supported",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.expected {
				t.Errorf("expected error message %q, got %q", test.expected, got)
			}
		})
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	if errors.Is(ErrNotFound, ErrNotSupported) {
		t.Error("ErrNotFound should not be the same as ErrNotSupported")
	}

	if errors.Is(ErrNotSupported, ErrNotFound) {
		t.Error("ErrNotSupported should not be the same as ErrNotFound")
	}
}

func TestErrorsCanBeWrapped(t *testing.T) {
	wrappedNotFound := errors.New("wrapped: " + ErrNotFound.Error())
	wrappedNotSupported := errors.New("wrapped: " + ErrNotSupported.Error())

	if wrappedNotFound.Error() != "wrapped: resource not found" {
		t.Errorf("unexpected wrapped error message: %s", wrappedNotFound.Error())
	}

	if wrappedNotSupported.Error() != "wrapped: not supported" {
		t.Errorf("unexpected wrapped error message: %s", wrappedNotSupported.Error())
	}
}
