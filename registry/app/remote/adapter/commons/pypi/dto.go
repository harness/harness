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

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	extensions = []string{
		".tar.gz",
		".tar.bz2",
		".tar.xz",
		".zip",
		".whl",
		".egg",

		".exe",
		".app",
		".dmg",
	}

	exeRegex = regexp.MustCompile(`(\d+(?:\.\d+)+)`)
)

type SimpleMetadata struct {
	Title    string
	MetaName string
	Content  string
	Packages []Package
}

type Package struct {
	Name  string
	ATags map[string]string
}

// URL returns the "href" attribute from the package's map.
func (p Package) URL() string {
	return p.ATags["href"]
}
func (p Package) Valid() bool {
	return p.URL() != "" && p.Name != ""
}

// RequiresPython returns the "data-requires-python" attribute (unescaped) from the package's map.
func (p Package) RequiresPython() string {
	val := p.ATags["data-requires-python"]
	// unescape HTML entities like "&gt;"
	return html.UnescapeString(val)
}

// Version Fetches version from format:
// The wheel filename is {distribution}-{version}(-{build tag})?-{python tag}-{abi tag}-{platform tag}.whl
// SRC: https://packaging.python.org/en/latest/specifications/binary-distribution-format/#file-name-convention
func (p Package) Version() string {
	return GetPyPIVersion(p.Name)
}

func (p Package) String() string {
	return fmt.Sprintf("Name: %s, Version: %s, URL: %s, RequiresPython: %s", p.Name, p.Version(), p.URL(),
		p.RequiresPython())
}

func GetPyPIVersion(filename string) string {
	base, ext, err := stripRecognizedExtension(filename)
	if err != nil {
		return ""
	}

	splits := strings.Split(base, "-")
	if len(splits) < 2 {
		return ""
	}

	switch ext {
	case ".whl", ".egg":
		return splits[1]
	case ".tar.gz", ".tar.bz2", ".tar.xz", ".zip", ".dmg", ".app":
		return splits[len(splits)-1]
	case ".exe":
		match := exeRegex.FindStringSubmatch(filename)
		if len(match) > 1 {
			return match[1]
		}
		return splits[len(splits)-1]
	default:
		return ""
	}
}

func stripRecognizedExtension(filename string) (string, string, error) {
	for _, x := range extensions {
		if strings.HasSuffix(strings.ToLower(filename), x) {
			base := filename[:len(filename)-len(x)]
			return base, x, nil
		}
	}

	return "", "", errors.New("unrecognized file extension")
}
