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

package insights

import (
	"strconv"
	"strings"
)

// parseSemver parses "1.2.3", "v1.2", "app@1.2.3+build" or "1.2.3-rc1" into numeric
// components. Pre-release and build suffixes are ignored. ok is false if the
// release does not look like a version.
func parseSemver(release string) ([3]int, bool) {
	var out [3]int
	if i := strings.LastIndex(release, "@"); i >= 0 {
		release = release[i+1:]
	}
	release = strings.TrimPrefix(strings.TrimPrefix(release, "v"), "V")
	if i := strings.IndexAny(release, "-+"); i >= 0 {
		release = release[:i]
	}
	parts := strings.Split(release, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// compareSemver returns -1, 0 or 1; ok is false if either side is not a version.
func compareSemver(a, b string) (int, bool) {
	va, okA := parseSemver(a)
	vb, okB := parseSemver(b)
	if !okA || !okB {
		return 0, false
	}
	for i := range va {
		switch {
		case va[i] < vb[i]:
			return -1, true
		case va[i] > vb[i]:
			return 1, true
		}
	}
	return 0, true
}
