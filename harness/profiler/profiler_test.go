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

package profiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseType(t *testing.T) {
	var tests = []struct {
		raw          string
		expectedType Type
		expectedOk   bool
	}{
		// basic invalid tests
		{"", Type(""), false},
		{"a", Type(""), false},
		{"g cp", Type(""), false},

		// ensure case insensitivity
		{"gcp", TypeGCP, true},
		{"GCP", TypeGCP, true},

		// ensure trim space works
		{" gcp ", TypeGCP, true},
		{"	GCP	", TypeGCP, true},

		// testing all valid values
		{"gcp", TypeGCP, true},
	}

	for i, test := range tests {
		parsedType, ok := ParseType(test.raw)

		assert.Equal(t, test.expectedOk, ok, "test case %d with input '%s'", i, test.raw)
		assert.Equal(t, test.expectedType, parsedType, "test case %d with input '%s'", i, test.raw)
	}
}
