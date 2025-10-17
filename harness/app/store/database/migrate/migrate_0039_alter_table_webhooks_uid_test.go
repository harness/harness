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

package migrate

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookDisplayNameToIdentifier(t *testing.T) {
	var tests = []struct {
		displayName        string
		randomize          bool
		expectedIdentifier string
		expectedRandomized bool
	}{
		// ensure only allowed characters get through
		{"az.A-Z_09", false, "az.A-Z_09", false},
		{"a" + string(rune(21)) + "a", false, "aa", false},
		{"a a", false, "a_a", false},

		// ensure leading/trailing special characters are removed
		{".-_a_-.", false, "a", false},

		// doesn't start with numbers
		{"0", false, "_0", false},

		// consecutive special characters are removed
		{"a .-__--..a", false, "a_a", false},

		// max length requirements
		{strings.Repeat("a", 101), false, strings.Repeat("a", 100), false},
		{" " + strings.Repeat("a", 100) + " ", false, strings.Repeat("a", 100), false},

		// empty identifier after sanitization
		{"", false, "webhook", true},
		{string(rune(21)), false, "webhook", true},
		{" .-_-. ", false, "webhook", true},

		// randomized
		{"a", true, "a", true},
		{strings.Repeat("a", 100), true, strings.Repeat("a", 95), true},

		// smoke tests
		{"harness (pipeline NR. #1)", false, "harness_pipeline_NR.1", false},
		{".harness/pipeline/Build.yaml", true, "harness_pipeline_Build.yaml", true},
		{".harness/pipeline/Build.yaml", true, "harness_pipeline_Build.yaml", true},
	}

	rndSuffixRegex := regexp.MustCompile("^_[a-z0-9]{4}$")

	for i, test := range tests {
		identifier, err := WebhookDisplayNameToIdentifier(test.displayName, test.randomize)
		assert.NoError(t, err, "test case %d - migration ended in error (unexpected)") // no errors expected

		if test.expectedRandomized {
			assert.True(t, len(identifier) >= 5, "test case %d - identifier length doesn't indicate random suffix", i)

			rndSuffix := identifier[len(identifier)-5:]
			identifier = identifier[:len(identifier)-5]

			matched := rndSuffixRegex.Match([]byte(rndSuffix))
			assert.True(t, matched, "test case %d - identifier doesn't contain expected random suffix", i)
		}

		assert.Equal(t, test.expectedIdentifier, identifier, "test case %d doesn't match the expected identifier'", i)
	}
}
