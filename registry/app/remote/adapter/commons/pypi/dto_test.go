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

package pypi

import "testing"

func TestGetPyPIVersion(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		// Wheel files (.whl): {distribution}-{version}-{python}-{abi}-{platform}.whl
		{name: "wheel simple", filename: "numpy-1.24.0-cp311-cp311-manylinux_2_17_x86_64.whl", want: "1.24.0"},
		{name: "wheel with build tag", filename: "package-2.0.1-1-cp39-cp39-win_amd64.whl", want: "2.0.1"},
		{name: "wheel namespaced package", filename: "zope.interface-5.5.0-cp39-cp39-win_amd64.whl", want: "5.5.0"},
		{name: "wheel request package", filename: "requests-2.4.1-py2.py3-none-any.whl", want: "2.4.1"},
		{name: "wheel request package with beta", filename: "requests-2.4.1-py2.py3-none-any.whl", want: "2.4.1"},

		// Egg files (.egg): {name}-{version}(-{pyver}).egg
		{name: "egg simple", filename: "mypackage-1.0.egg", want: "1.0"},
		{name: "egg with pyver", filename: "mypackage-1.0-py3.9.egg", want: "1.0"},

		// Source distributions (.tar.gz, .zip, etc.): {name}-{version}.ext
		{name: "tar.gz simple", filename: "numpy-1.24.0.tar.gz", want: "1.24.0"},
		{name: "tar.gz hyphenated name", filename: "my-cool-package-2.0.tar.gz", want: "2.0"},
		{name: "tar.gz version with hyphen post", filename: "package-1.0-1.tar.gz", want: "1.0-1"},
		{name: "tar.gz version with pre-release", filename: "pymdptoolbox-4.0-b3.tar.gz", want: "4.0-b3"},
		{name: "tar.bz2", filename: "package-3.1.tar.bz2", want: "3.1"},
		{name: "tar.xz", filename: "package-0.9.1.tar.xz", want: "0.9.1"},
		{name: "zip", filename: "mylib-1.2.3.zip", want: "1.2.3"},
		{name: "tar.gz v prefix version", filename: "package-v1.0.tar.gz", want: "v1.0"},
		{name: "tar.gz numeric name prefix", filename: "3to2-1.1.tar.gz", want: "1.1"},
		{name: "tar.gz name with digits", filename: "py3-package-1.0.tar.gz", want: "1.0"},

		// Exe files (.exe)
		{name: "exe with version", filename: "package-1.2.3.exe", want: "1.2.3"},

		// Edge cases.
		{name: "unrecognized extension", filename: "package-1.0.unknown", want: ""},
		{name: "no hyphen", filename: "package.tar.gz", want: ""},
		{name: "empty filename", filename: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPyPIVersion(tt.filename)
			if got != tt.want {
				t.Errorf("GetPyPIVersion(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}
