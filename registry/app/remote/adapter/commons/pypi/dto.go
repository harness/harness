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
	"net/url"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
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
	SimpleURL string
	Name      string
	ATags     map[string]string
}

// URL returns the "href" attribute from the package's map.
func (p Package) URL() string {
	href := p.ATags["href"]
	parsedURL, err := url.Parse(href)
	if err != nil {
		return href
	}

	if parsedURL.IsAbs() {
		return href
	}

	// If href is relative, resolve it against SimpleURL
	baseURL, err := url.Parse(p.SimpleURL)
	if err != nil {
		log.Err(err).Msgf("failed to parse url %s", p.SimpleURL)
		return href
	}

	return baseURL.ResolveReference(parsedURL).String()
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

// Version extracts the version from the package filename.
// packageName is the distribution name used to find the exact name-version split point.
func (p Package) Version(packageName ...string) string {
	return GetPyPIVersion(p.Name, packageName...)
}

func (p Package) String() string {
	return fmt.Sprintf("Name: %s, Version: %s, URL: %s, RequiresPython: %s", p.Name, p.Version(), p.URL(),
		p.RequiresPython())
}

// GetPyPIVersion extracts the version from a Python package filename.
// Following pip's approach: when packageName is provided, it canonicalizes both
// the name and filename segments to find the exact split point between name and version.
// When packageName is empty, it falls back to heuristic-based parsing.
// Reference: https://github.com/pypa/pip/blob/main/src/pip/_internal/index/package_finder.py
func GetPyPIVersion(filename string, packageName ...string) string {
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
		if len(packageName) > 0 && packageName[0] != "" {
			if idx := findNameVersionSep(base, packageName[0]); idx >= 0 {
				version := base[idx+1:]
				// Verify the version part starts with a digit or 'v' prefix.
				// Filenames like "package-docs-1.0.tar.gz" would give "docs-1.0"
				// which should fall through to the heuristic.
				if len(version) > 0 && (version[0] >= '0' && version[0] <= '9' ||
					((version[0] == 'v' || version[0] == 'V') && len(version) > 1 && version[1] >= '0' && version[1] <= '9')) {
					return version
				}
			}
		}
		// Fallback: find first digit-starting segment.
		if idx := findVersionStart(splits); idx > 0 {
			return strings.Join(splits[idx:], "-")
		}
		return splits[len(splits)-1]
	case ".exe":
		// For .exe files, prefer regex extraction since filenames contain platform
		// tags (e.g., "package-1.0.1.win32-py2.3.exe") that findNameVersionSep
		// would incorrectly include in the version.
		match := exeRegex.FindStringSubmatch(filename)
		if len(match) > 1 {
			return match[1]
		}
		if idx := findVersionStart(splits); idx > 0 {
			return strings.Join(splits[idx:], "-")
		}
		return splits[len(splits)-1]
	default:
		return ""
	}
}

// findNameVersionSep finds the dash index that separates the package name from the version,
// following pip's approach: iterate through each '-' and check if the prefix, when
// canonicalized, matches the canonical package name.
func findNameVersionSep(stem, packageName string) int {
	canonicalName := canonicalizeName(packageName)
	for i, c := range stem {
		if c != '-' {
			continue
		}
		if canonicalizeName(stem[:i]) == canonicalName {
			return i
		}
	}
	return -1
}

// canonicalizeName normalizes a Python package name per PEP 503:
// lowercase, and replace any run of [-_.] with a single dash.
func canonicalizeName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	prevSep := false
	for _, c := range strings.ToLower(name) {
		if c == '-' || c == '_' || c == '.' {
			if !prevSep {
				b.WriteByte('-')
				prevSep = true
			}
			continue
		}
		prevSep = false
		b.WriteRune(c)
	}
	return b.String()
}

// findVersionStart returns the index of the first segment (starting from index 1)
// that begins with a digit, indicating the start of a version string.
func findVersionStart(segments []string) int {
	for i := 1; i < len(segments); i++ {
		if len(segments[i]) > 0 && segments[i][0] >= '0' && segments[i][0] <= '9' {
			return i
		}
	}
	return -1
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
