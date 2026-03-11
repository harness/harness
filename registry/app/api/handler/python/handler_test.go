//  Copyright 2023 Harness, Inc.
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

package python

import "testing"

func TestIsValidNameAndVersion(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		version string
		want    bool
	}{
		// Valid names and versions.
		{name: "simple name and version", image: "mypackage", version: "1.0.0", want: true},
		{name: "single char name", image: "a", version: "1", want: true},
		{name: "name with hyphens", image: "my-package", version: "1.0", want: true},
		{name: "name with dots", image: "my.package", version: "1.0", want: true},
		{name: "name with underscores", image: "my_package", version: "1.0", want: true},
		{name: "name with mixed separators", image: "my-pkg.test_v2", version: "1.0", want: true},
		{name: "numeric name", image: "123", version: "1.0", want: true},
		{name: "alphanumeric name", image: "pkg123abc", version: "1.0", want: true},

		// Valid version formats (PEP 440).
		{name: "version with v prefix", image: "pkg", version: "v1.0.0", want: true},
		{name: "version with epoch", image: "pkg", version: "1!2.0", want: true},
		{name: "version with pre-release alpha", image: "pkg", version: "1.0a1", want: true},
		{name: "version with pre-release beta", image: "pkg", version: "1.0b2", want: true},
		{name: "version with pre-release rc", image: "pkg", version: "1.0rc1", want: true},
		{name: "version with pre-release preview", image: "pkg", version: "1.0.preview1", want: true},
		{name: "version with post release", image: "pkg", version: "1.0.post1", want: true},
		{name: "version with dev release", image: "pkg", version: "1.0.dev3", want: true},
		{name: "version with local segment", image: "pkg", version: "1.0+local.1", want: true},
		{name: "complex version", image: "pkg", version: "1!2.3.4a5.post6.dev7+local.8", want: true},
		{name: "version with hyphen post", image: "pkg", version: "1.0-1", want: true},
		{name: "version with pre no number", image: "pkg", version: "1.0a", want: true},
		{name: "version with underscore separator", image: "pkg", version: "1.0_dev1", want: true},
		{name: "version with hyphen separator", image: "pkg", version: "1.0-dev1", want: true},

		{name: "version with non semvar version", image: "pymdptoolbox", version: "4.0b1", want: true},
		{name: "version with non semvar version", image: "pymdptoolbox", version: "4.0-b3", want: true},

		// Invalid names.
		{name: "empty name", image: "", version: "1.0", want: false},
		{name: "name starting with dot", image: ".pkg", version: "1.0", want: false},
		{name: "name ending with dot", image: "pkg.", version: "1.0", want: false},
		{name: "name starting with hyphen", image: "-pkg", version: "1.0", want: false},
		{name: "name ending with hyphen", image: "pkg-", version: "1.0", want: false},
		{name: "name starting with underscore", image: "_pkg", version: "1.0", want: false},
		{name: "name ending with underscore", image: "pkg_", version: "1.0", want: false},
		{name: "name with spaces", image: "my package", version: "1.0", want: false},
		{name: "name with special chars", image: "pkg@1", version: "1.0", want: false},

		// Invalid versions.
		{name: "empty version", image: "pkg", version: "", want: false},
		{name: "version with only letters", image: "pkg", version: "abc", want: false},
		{name: "version with spaces", image: "pkg", version: "1.0 0", want: false},
		{name: "version starting with dot", image: "pkg", version: ".1.0", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNameAndVersion(tt.image, tt.version)
			if got != tt.want {
				t.Errorf("isValidNameAndVersion(%q, %q) = %v, want %v", tt.image, tt.version, got, tt.want)
			}
		})
	}
}
