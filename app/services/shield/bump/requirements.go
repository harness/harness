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

package bump

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reqLineRe = regexp.MustCompile(
		`^(\s*)([A-Za-z0-9][A-Za-z0-9._-]*)(\s*\[[^\]]*\])?(\s*)==(\s*)([^\s;#\\]+)(.*)$`)
	pyNameSepRe = regexp.MustCompile(`[-_.]+`)
)

// normalizePyName applies PEP 503 name normalization.
func normalizePyName(n string) string {
	return strings.ToLower(pyNameSepRe.ReplaceAllString(n, "-"))
}

// BumpRequirements replaces the `name==from` pin by `name==to` in a requirements.txt. Files using
// hashes cannot be updated (the hashes of the new version are unknown) and need the agent.
func BumpRequirements(content []byte, name, from, to string) ([]byte, error) {
	text := string(content)
	if strings.Contains(text, "--hash") || strings.Contains(text, "--require-hashes") {
		return nil, needsAgent("requirements file uses --hash pins")
	}
	lines := strings.Split(text, "\n")
	want := normalizePyName(name)
	changed := false
	for i, l := range lines {
		m := reqLineRe.FindStringSubmatch(strings.TrimRight(l, "\r"))
		if m == nil || normalizePyName(m[2]) != want {
			continue
		}
		if m[6] != from {
			return nil, needsAgent("%s is pinned to %s, expected %s", name, m[6], from)
		}
		eol := ""
		if strings.HasSuffix(l, "\r") {
			eol = "\r"
		}
		lines[i] = m[1] + m[2] + m[3] + m[4] + "==" + m[5] + to + m[7] + eol
		changed = true
	}
	if !changed {
		return nil, fmt.Errorf("%w: no pin of %s==%s found", ErrUnsupported, name, from)
	}
	return []byte(strings.Join(lines, "\n")), nil
}
