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

package parser

import (
	"testing"
	"time"

	"github.com/harness/gitness/types"
)

func TestObjectHeaderIdentity(t *testing.T) {
	tzBelgrade, _ := time.LoadLocation("Europe/Belgrade")
	tzCalifornia, _ := time.LoadLocation("America/Los_Angeles")

	tests := []struct {
		name     string
		input    string
		expected types.Signature
	}{
		{
			name:  "test1",
			input: "Vincent Willem van Gogh <van.gogh@harness.io> 1748779200 +0200",
			expected: types.Signature{
				Identity: types.Identity{Name: "Vincent Willem van Gogh", Email: "van.gogh@harness.io"},
				When:     time.Date(2025, time.June, 1, 14, 0, 0, 0, tzBelgrade),
			},
		},
		{
			name:  "test2",
			input: "徳川家康 <tokugawa@harness.io> 1748779200 -0700",
			expected: types.Signature{
				Identity: types.Identity{Name: "徳川家康", Email: "tokugawa@harness.io"},
				When:     time.Date(2025, time.June, 1, 5, 0, 0, 0, tzCalifornia),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, email, when, err := ObjectHeaderIdentity(test.input)
			if err != nil {
				t.Error(err)
				return
			}

			if test.expected.Identity.Name != name {
				t.Errorf("name mismatch - expected: %v, got: %v", test.expected.Identity.Name, name)
			}
			if test.expected.Identity.Email != email {
				t.Errorf("email mismatch - expected: %v, got: %v", test.expected.Identity.Email, email)
			}
			if !test.expected.When.Equal(when) {
				t.Errorf("timestamp mismatch - expected: %s, got: %s",
					test.expected.When.Format(time.RFC3339), when.Format(time.RFC3339))
			}
		})
	}
}
