// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
