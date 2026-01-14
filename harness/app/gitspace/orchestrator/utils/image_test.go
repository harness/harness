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

package utils

import (
	"testing"
)

func TestCheckContainerImageExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Valid expressions
		{"nginx", ExpressionTypeRepository},
		{"nginx:latest", ExpressionTypeImageTag},
		{"nginx@sha256:01eb582bca526c37aad8dbc5c9ba69899ecf2540f561694979d153bcbbf146fe",
			ExpressionTypeImageDigest},
		{"repo/*", ExpressionTypeWildcardRepo},
		{"repo/image:*", ExpressionTypeWildcardTag},
		{"repo/image:dev*", ExpressionTypeWildcardTagPrefix},
		{"repo/image*", ExpressionTypeWildcardRepoPrefix},

		// Invalid expressions
		{"nginx:*:latest", ExpressionTypeInvalid},
		{"nginx:*latest", ExpressionTypeInvalid},
		{"nginx:lat*est", ExpressionTypeInvalid},
		{"*nginx:latest", ExpressionTypeInvalid},
		{"invalid@@@", ExpressionTypeInvalid},
		{"repo/image:!invalid", ExpressionTypeInvalid},
		{"repo/image:prefix*", ExpressionTypeWildcardTagPrefix},
		{"mcr.microsoft.com/devcontainers/", ExpressionTypeInvalid},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			actual := CheckContainerImageExpression(test.input)
			if actual != test.expected {
				t.Errorf("CheckContainerImageExpression(%q) = %q, want %q", test.input, actual, test.expected)
			}
		})
	}
}

func TestMatchesContainerImageExpression(t *testing.T) {
	tests := []struct {
		expr     string
		image    string
		expected bool
	}{
		// Exact image tag
		{"nginx:latest", "nginx:latest", true},
		{"nginx:1.19", "nginx:latest", false},

		// Image digest
		{"nginx@sha256:01eb582bca526c37aad8dbc5c9ba69899ecf2540f561694979d153bcbbf146fe",
			"nginx@sha256:01eb582bca526c37aad8dbc5c9ba69899ecf2540f561694979d153bcbbf146fe", true},
		{"nginx@sha256:abc123", "nginx@sha256:def456", false},

		// Repository match
		{"nginx", "nginx:latest", true},
		{"nginx", "nginx@sha256:abc123", true},
		{"nginx", "nginx", true},

		// Wildcard repo
		{"repo/*", "repo/app:tag", true},
		{"repo/*", "repo/sub/app:tag", true},
		{"repo/*", "other/image:tag", false},

		// Wildcard tag
		{"repo/image:*", "repo/image:latest", true},
		{"repo/image:*", "repo/image:v1.0", true},
		{"repo/image:*", "repo/image", false},
		{"repo/image:*", "repo/image@sha256:abc", false},

		// Wildcard tag prefix
		{"repo/image:dev*", "repo/image:dev123", true},
		{"repo/image:dev*", "repo/image:dev", true},
		{"repo/image:dev*", "repo/image:prod", false},
		{"repo/image:dev*", "other/image:dev123", false},

		// Wildcard repo prefix
		{"repo/image*", "repo/image:tag", true},
		{"repo/image*", "repo/image-extra:tag", true},
		{"repo/image*", "repo/image/child:tag", true},
		{"repo/image*", "other/image:tag", false},

		// Invalid expression
		{"nginx:*latest", "nginx:latest", false},
		{"mcr.microsoft.com/devcontainers/", "mcr.microsoft.com/devcontainers/python:latest", false},

		// Valid wildcard repo correction
		{"mcr.microsoft.com/devcontainers/*",
			"mcr.microsoft.com/devcontainers/python:latest", true},
		{"mcr.microsoft.com/devcontainers/",
			"mcr.microsoft.com/devcontainers/python:latest", false},
		{"mcr.microsoft.com/devcontainers/python",
			"mcr.microsoft.com/devcontainers/python:latest", true},
		{"mcr.microsoft.com/devcontainers/python/",
			"mcr.microsoft.com/devcontainers/python:latest", false},
		{"mcr.microsoft.com/devcontainers/python/*",
			"mcr.microsoft.com/devcontainers/python:latest", false},
		{"mcr.microsoft.com/devcontainers/python/*",
			"mcr.microsoft.com/devcontainers/python/image:latest", true},
		{"mcr.microsoft.com/devcontainers/python:*",
			"mcr.microsoft.com/devcontainers/python:latest", true},
	}

	for _, test := range tests {
		name := test.expr + " matches " + test.image
		t.Run(name, func(t *testing.T) {
			actual := MatchesContainerImageExpression(test.expr, test.image)
			if actual != test.expected {
				t.Errorf("MatchesContainerImageExpression(%q, %q) = %v, want %v", test.expr, test.image, actual, test.expected)
			}
		})
	}
}
