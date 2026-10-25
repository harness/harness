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
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrInvalidVersion is returned for versions a Comparer cannot parse.
var ErrInvalidVersion = errors.New("osv: invalid version")

// Comparer orders versions of one versioning scheme.
type Comparer interface {
	// Compare returns -1, 0 or 1.
	Compare(a, b string) (int, error)
	// Line returns the compatibility line of v (the major version for
	// semver, epoch plus first release segment for PEP 440).
	Line(v string) (string, error)
}

// ComparerFor returns the comparer for an OSV ecosystem, or nil if the
// ecosystem is not supported for ECOSYSTEM-typed ranges.
func ComparerFor(ecosystem string) Comparer {
	switch ecosystem {
	case EcosystemGo, EcosystemNPM:
		return SemverComparer{}
	case EcosystemPyPI:
		return PEP440Comparer{}
	}
	return nil
}

// ---------------------------------------------------------------------
// Semver

// SemverComparer implements Semantic Versioning 2.0 precedence. It accepts
// an optional leading "v" (Go modules, "v1.2.3"), ignores build metadata
// (so Go's "+incompatible" is dropped) and tolerates missing minor/patch
// components ("1.2" == "1.2.0"), which appear in OSV "fixed" events.
// Go pseudo-versions (v0.0.0-20210101000000-abcdef123456) are ordinary
// pre-releases of their base version, which orders them correctly relative
// to other pseudo-versions and tagged releases.
type SemverComparer struct{}

type semver struct {
	core [3]string // normalised decimal strings without leading zeros
	pre  []string
}

var numRe = regexp.MustCompile(`^[0-9]+$`)

func parseSemver(s string) (semver, error) {
	orig := s
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "=")
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var v semver
	core, pre, hasPre := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return v, fmt.Errorf("%w: %q", ErrInvalidVersion, orig)
	}
	for i := 0; i < 3; i++ {
		if i >= len(parts) {
			v.core[i] = "0"
			continue
		}
		if !numRe.MatchString(parts[i]) {
			return v, fmt.Errorf("%w: %q", ErrInvalidVersion, orig)
		}
		v.core[i] = trimZeros(parts[i])
	}
	if hasPre {
		if pre == "" {
			return v, fmt.Errorf("%w: %q", ErrInvalidVersion, orig)
		}
		v.pre = strings.Split(pre, ".")
		for _, id := range v.pre {
			if id == "" {
				return v, fmt.Errorf("%w: %q", ErrInvalidVersion, orig)
			}
		}
	}
	return v, nil
}

func trimZeros(s string) string {
	s = strings.TrimLeft(s, "0")
	if s == "" {
		return "0"
	}
	return s
}

func cmpNumStr(a, b string) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

func (SemverComparer) Compare(a, b string) (int, error) {
	va, err := parseSemver(a)
	if err != nil {
		return 0, err
	}
	vb, err := parseSemver(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		if c := cmpNumStr(va.core[i], vb.core[i]); c != 0 {
			return c, nil
		}
	}
	switch {
	case len(va.pre) == 0 && len(vb.pre) == 0:
		return 0, nil
	case len(va.pre) == 0:
		return 1, nil
	case len(vb.pre) == 0:
		return -1, nil
	}
	for i := 0; i < len(va.pre) && i < len(vb.pre); i++ {
		x, y := va.pre[i], vb.pre[i]
		xn, yn := numRe.MatchString(x), numRe.MatchString(y)
		var c int
		switch {
		case xn && yn:
			c = cmpNumStr(trimZeros(x), trimZeros(y))
		case xn:
			c = -1
		case yn:
			c = 1
		default:
			c = strings.Compare(x, y)
		}
		if c != 0 {
			return c, nil
		}
	}
	switch {
	case len(va.pre) < len(vb.pre):
		return -1, nil
	case len(va.pre) > len(vb.pre):
		return 1, nil
	}
	return 0, nil
}

func (SemverComparer) Line(v string) (string, error) {
	sv, err := parseSemver(v)
	if err != nil {
		return "", err
	}
	return sv.core[0], nil
}

// ---------------------------------------------------------------------
// PEP 440

// PEP440Comparer implements a subset of PEP 440 ordering: optional epoch
// ("1!"), numeric release segments (trailing zeros are insignificant),
// pre-releases (a/alpha, b/beta, c/rc/pre/preview), post-releases
// (.postN, -N, rev, r) and dev releases (.devN), with the usual
// separators ('.', '-', '_') and case-insensitivity, plus an optional
// leading "v".
//
// Limits: local version labels ("+ubuntu1") are ignored for ordering;
// legacy non-PEP 440 versions are rejected with ErrInvalidVersion; the
// implicit-zero shorthand "1.0post" is accepted but exotic spellings are
// not.
type PEP440Comparer struct{}

type pep440 struct {
	epoch   uint64
	release []uint64
	preKind int // 0 none, 1 a, 2 b, 3 rc
	preNum  uint64
	hasPost bool
	postNum uint64
	hasDev  bool
	devNum  uint64
}

var pep440Re = regexp.MustCompile(`^v?(?:(\d+)!)?(\d+(?:\.\d+)*)` +
	`(?:[-_.]?(alpha|a|beta|b|preview|pre|rc|c)[-_.]?(\d*))?` +
	`(?:(?:-(\d+))|(?:[-_.]?(post|rev|r)[-_.]?(\d*)))?` +
	`(?:[-_.]?(dev)[-_.]?(\d*))?` +
	`(?:\+[a-z0-9]+(?:[-_.][a-z0-9]+)*)?$`)

func atou(s string) uint64 {
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

func parsePEP440(s string) (pep440, error) {
	m := pep440Re.FindStringSubmatch(strings.ToLower(strings.TrimSpace(s)))
	if m == nil {
		return pep440{}, fmt.Errorf("%w: %q", ErrInvalidVersion, s)
	}
	var v pep440
	v.epoch = atou(m[1])
	for _, p := range strings.Split(m[2], ".") {
		v.release = append(v.release, atou(p))
	}
	for len(v.release) > 1 && v.release[len(v.release)-1] == 0 {
		v.release = v.release[:len(v.release)-1]
	}
	switch m[3] {
	case "alpha", "a":
		v.preKind = 1
	case "beta", "b":
		v.preKind = 2
	case "preview", "pre", "rc", "c":
		v.preKind = 3
	}
	if v.preKind != 0 {
		v.preNum = atou(m[4])
	}
	if m[5] != "" {
		v.hasPost, v.postNum = true, atou(m[5])
	} else if m[6] != "" {
		v.hasPost, v.postNum = true, atou(m[7])
	}
	if m[8] != "" {
		v.hasDev, v.devNum = true, atou(m[9])
	}
	return v, nil
}

func cmpU(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// rank orders the pre/post/dev markers as specified by PEP 440's sort key.
func (v pep440) key() (pre [2]uint64, post [2]uint64, dev [2]uint64) {
	// pre: dev-only releases sort before any pre-release; no pre-release
	// sorts after all pre-releases.
	switch {
	case v.preKind == 0 && !v.hasPost && v.hasDev:
		pre = [2]uint64{0, 0}
	case v.preKind == 0:
		pre = [2]uint64{5, 0}
	default:
		pre = [2]uint64{uint64(v.preKind), v.preNum}
	}
	if v.hasPost {
		post = [2]uint64{1, v.postNum}
	}
	if v.hasDev {
		dev = [2]uint64{0, v.devNum}
	} else {
		dev = [2]uint64{1, 0}
	}
	return
}

func (PEP440Comparer) Compare(a, b string) (int, error) {
	va, err := parsePEP440(a)
	if err != nil {
		return 0, err
	}
	vb, err := parsePEP440(b)
	if err != nil {
		return 0, err
	}
	if c := cmpU(va.epoch, vb.epoch); c != 0 {
		return c, nil
	}
	n := len(va.release)
	if len(vb.release) > n {
		n = len(vb.release)
	}
	for i := 0; i < n; i++ {
		var x, y uint64
		if i < len(va.release) {
			x = va.release[i]
		}
		if i < len(vb.release) {
			y = vb.release[i]
		}
		if c := cmpU(x, y); c != 0 {
			return c, nil
		}
	}
	pa, qa, da := va.key()
	pb, qb, db := vb.key()
	for _, pair := range [][2][2]uint64{{pa, pb}, {qa, qb}, {da, db}} {
		for i := 0; i < 2; i++ {
			if c := cmpU(pair[0][i], pair[1][i]); c != 0 {
				return c, nil
			}
		}
	}
	return 0, nil
}

func (PEP440Comparer) Line(v string) (string, error) {
	pv, err := parsePEP440(v)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d!%d", pv.epoch, pv.release[0]), nil
}
