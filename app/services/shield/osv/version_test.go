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

package osv

import (
	"errors"
	"testing"
)

func TestSemverCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"v1.2.3", "1.2.3", 0},
		{"1.2", "1.2.0", 0},
		{"1", "1.0.0", 0},
		{"1.2.3", "1.2.4", -1},
		{"1.10.0", "1.9.0", 1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"1.0.0-alpha.beta", "1.0.0-beta", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-beta.11", "1.0.0-rc.1", -1},
		{"1.0.0-rc.1", "1.0.0", -1},
		{"1.0.0+build1", "1.0.0+build2", 0},
		{"v2.0.0+incompatible", "2.0.0", 0},
		{"v99999999999999999999.0.0", "v9999999999999999999.0.0", 1},
		// Go pseudo-versions.
		{"v0.0.0-20210921155107-089bfa567519", "v0.0.0-20220314234659-1baeb1ce4c0b", -1},
		{"v0.0.0-20220314234659-1baeb1ce4c0b", "v0.1.0", -1},
		{"v1.2.4-0.20210101000000-abcdef123456", "v1.2.3", 1},
		{"v1.2.4-0.20210101000000-abcdef123456", "v1.2.4", -1},
	}
	var c SemverComparer
	for _, tc := range tests {
		got, err := c.Compare(tc.a, tc.b)
		if err != nil {
			t.Errorf("Compare(%q,%q): %v", tc.a, tc.b, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Compare(%q,%q)=%d want %d", tc.a, tc.b, got, tc.want)
		}
		if back, _ := c.Compare(tc.b, tc.a); back != -tc.want {
			t.Errorf("Compare(%q,%q)=%d not antisymmetric", tc.b, tc.a, back)
		}
	}
}

func TestSemverInvalid(t *testing.T) {
	var c SemverComparer
	for _, v := range []string{"", "abc", "1.2.3.4", "1.x.0", "1.0.0-", "1.0.0-a..b", "v"} {
		if _, err := c.Compare(v, "1.0.0"); !errors.Is(err, ErrInvalidVersion) {
			t.Errorf("%q: expected ErrInvalidVersion, got %v", v, err)
		}
	}
}

func TestPEP440Compare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0", "1.0.0", 0},
		{"1.0", "1", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.10", "1.9", 1},
		{"1.0.dev1", "1.0a1", -1},
		{"1.0a1", "1.0a2", -1},
		{"1.0a1", "1.0b1", -1},
		{"1.0b1", "1.0rc1", -1},
		{"1.0rc1", "1.0", -1},
		{"1.0", "1.0.post1", -1},
		{"1.0.post1", "1.0.post2", -1},
		{"1.0.post1", "1.0.1", -1},
		{"1.0a1.dev1", "1.0a1", -1},
		{"1.0.post1.dev1", "1.0.post1", -1},
		{"1.0.post1.dev1", "1.0", 1},
		{"1.0.dev1", "1.0", -1},
		{"1.0.dev1", "0.9", 1},
		{"1!0.5", "2.0", 1},
		{"1.0-1", "1.0.post1", 0},
		{"1.0ALPHA1", "1.0a1", 0},
		{"1.0c1", "1.0rc1", 0},
		{"1.0-rc.1", "1.0rc1", 0},
		{"v1.0", "1.0", 0},
		{"1.0+local.1", "1.0", 0},
		{"2.2.20", "2.2", 1},
		{"3.0.14", "3.0.9", 1},
	}
	var c PEP440Comparer
	for _, tc := range tests {
		got, err := c.Compare(tc.a, tc.b)
		if err != nil {
			t.Errorf("Compare(%q,%q): %v", tc.a, tc.b, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Compare(%q,%q)=%d want %d", tc.a, tc.b, got, tc.want)
		}
		if back, _ := c.Compare(tc.b, tc.a); back != -tc.want {
			t.Errorf("Compare(%q,%q)=%d not antisymmetric", tc.b, tc.a, back)
		}
	}
}

func TestPEP440Invalid(t *testing.T) {
	var c PEP440Comparer
	for _, v := range []string{"", "latest", "1.0.*", "1..0", "abc1"} {
		if _, err := c.Compare(v, "1.0"); !errors.Is(err, ErrInvalidVersion) {
			t.Errorf("%q: expected ErrInvalidVersion, got %v", v, err)
		}
	}
}

func TestLine(t *testing.T) {
	if l, _ := (SemverComparer{}).Line("v2.3.4"); l != "2" {
		t.Errorf("semver line %q", l)
	}
	if l, _ := (PEP440Comparer{}).Line("1!3.2"); l != "1!3" {
		t.Errorf("pep440 line %q", l)
	}
}

func TestComparerFor(t *testing.T) {
	if ComparerFor(EcosystemGo) == nil || ComparerFor(EcosystemNPM) == nil || ComparerFor(EcosystemPyPI) == nil {
		t.Fatal("missing comparer")
	}
	if ComparerFor("Debian:11") != nil {
		t.Fatal("unexpected comparer")
	}
}
