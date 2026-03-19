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
		{name: "tar.gz name with v2", filename: "gax-google-logging-v2-0.8.3.tar.gz", want: "0.8.3"},

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

func TestGetPyPIVersionWithPackageName(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		packageName string
		want        string
	}{
		// Package name resolves ambiguous sdist filenames.
		{name: "name with v2 segment", filename: "gax-google-logging-v2-0.8.3.tar.gz",
			packageName: "gax-google-logging-v2", want: "0.8.3"},
		{name: "UUID-like name", filename: "01d61084-d29e-11e9-96d1-7c5cf84ffe8e-0.1.0.tar.gz",
			packageName: "01d61084-d29e-11e9-96d1-7c5cf84ffe8e", want: "0.1.0"},
		{name: "name with version-like digits", filename: "101hello-0.0.1-redish101-0.0.2.tar.gz",
			packageName: "101hello-0.0.1-redish101", want: "0.0.2"},
		{name: "numeric name prefix", filename: "3-1-1.0.0.zip",
			packageName: "3-1", want: "1.0.0"},
		{name: "underscore to dash normalization", filename: "my_cool_package-2.0.tar.gz",
			packageName: "my-cool-package", want: "2.0"},
		{name: "dot normalization", filename: "zope.interface-5.5.0.tar.gz",
			packageName: "zope.interface", want: "5.5.0"},
		{name: "mixed case", filename: "MyPackage-1.0.tar.gz",
			packageName: "mypackage", want: "1.0"},
		{name: "simple sdist", filename: "requests-2.31.0.tar.gz",
			packageName: "requests", want: "2.31.0"},
		{name: "hyphenated name", filename: "my-cool-package-2.0.tar.gz",
			packageName: "my-cool-package", want: "2.0"},

		// Wheel files ignore packageName (fixed format).
		{name: "wheel ignores package name", filename: "numpy-1.24.0-cp311-cp311-manylinux_2_17_x86_64.whl",
			packageName: "numpy", want: "1.24.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPyPIVersion(tt.filename, tt.packageName)
			if got != tt.want {
				t.Errorf("GetPyPIVersion(%q, %q) = %q, want %q", tt.filename, tt.packageName, got, tt.want)
			}
		})
	}
}
