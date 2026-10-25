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
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Finding is a vulnerability that affects an installed package.
type Finding struct {
	Package Package  `json:"package"`
	VulnID  string   `json:"vuln_id"`
	Aliases []string `json:"aliases,omitempty"`
	// CVEs are the CVE identifiers among the ID and aliases.
	CVEs    []string `json:"cves,omitempty"`
	Summary string   `json:"summary,omitempty"`

	// CVSSVector is the CVSS vector (v3.x preferred, else v4), if any.
	CVSSVector string `json:"cvss_vector,omitempty"`
	// CVSSScore is the base score. It is meaningful only if HasScore:
	// computed for v3.x vectors, or taken from a numeric OSV score.
	CVSSScore float64 `json:"cvss_score,omitempty"`
	HasScore  bool    `json:"has_score,omitempty"`
	// Severity is the qualitative rating: the database's own label when
	// present (e.g. GHSA "HIGH"), else derived from CVSSScore.
	Severity string `json:"severity,omitempty"`

	// FixedVersions are all distinct "fixed" versions of the matching
	// affected entries, sorted ascending.
	FixedVersions []string `json:"fixed_versions,omitempty"`
	// MinFixedVersion is the lowest fixed version greater than the installed
	// one, preferring the installed major line (see Line of the Comparer).
	// Empty if no fix is known.
	MinFixedVersion string `json:"min_fixed_version,omitempty"`
	// FixCrossLine is true if MinFixedVersion is in a different major line
	// than the installed version, i.e. no fix exists on the installed line.
	FixCrossLine bool `json:"fix_cross_line,omitempty"`
}

// Skipped reports a package that could not be evaluated.
type Skipped struct {
	Package Package `json:"package"`
	Reason  string  `json:"reason"`
}

// Result is the output of Match.
type Result struct {
	Findings []Finding `json:"findings"`
	Skipped  []Skipped `json:"skipped,omitempty"`
}

// Match evaluates packages against the database. Findings are ordered by
// package then vulnerability id. A vulnerability is reported once per
// package even if several affected entries or ranges match. GIT ranges are
// ignored, as are ECOSYSTEM ranges of ecosystems without a Comparer.
func Match(db *DB, pkgs []Package) *Result {
	res := &Result{}
	for _, p := range pkgs {
		entries := db.lookup(p.Ecosystem, p.Name)
		if len(entries) == 0 {
			continue
		}
		cmp := ComparerFor(p.Ecosystem)
		if cmp == nil {
			cmp = SemverComparer{}
		}
		// An unparseable installed version can still match explicit
		// "versions" lists by string equality; ranges are then ignored.
		_, verr := cmp.Compare(p.Version, p.Version)
		matched := 0
		byVuln := map[string]*matchAcc{}
		for _, e := range entries {
			a := &e.rec.Affected[e.idx]
			ok, fixed := affects(a, p, cmp)
			if !ok {
				continue
			}
			acc := byVuln[e.rec.ID]
			if acc == nil {
				acc = &matchAcc{rec: e.rec}
				byVuln[e.rec.ID] = acc
			}
			acc.affected = append(acc.affected, a)
			acc.fixed = append(acc.fixed, fixed...)
		}
		ids := make([]string, 0, len(byVuln))
		for id := range byVuln {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			res.Findings = append(res.Findings, buildFinding(p, byVuln[id], cmp))
			matched++
		}
		if verr != nil && matched == 0 {
			res.Skipped = append(res.Skipped, Skipped{Package: p, Reason: "unparseable version: " + p.Version})
		}
	}
	return res
}

type matchAcc struct {
	rec      *Record
	affected []*Affected
	fixed    []string
}

// affects reports whether version of p is affected by a, and returns the
// fixed versions of the supported ranges of a.
func affects(a *Affected, p Package, cmp Comparer) (bool, []string) {
	hit := false
	for _, v := range a.Versions {
		if v == p.Version {
			hit = true
			break
		}
		if c, err := cmp.Compare(v, p.Version); err == nil && c == 0 {
			hit = true
			break
		}
	}
	var fixed []string
	for _, r := range a.Ranges {
		var rc Comparer
		switch r.Type {
		case RangeSemver:
			rc = SemverComparer{}
		case RangeEcosystem:
			rc = ComparerFor(p.Ecosystem)
		}
		if rc == nil {
			continue // GIT or unsupported
		}
		in, err := inRange(r.Events, p.Version, rc)
		if err != nil {
			continue
		}
		hit = hit || in
		for _, ev := range r.Events {
			if ev.Fixed != "" {
				fixed = append(fixed, ev.Fixed)
			}
		}
	}
	return hit, fixed
}

type sortedEvent struct {
	kind    byte // 'i' introduced, 'f' fixed, 'l' last_affected
	version string
}

// inRange implements the OSV range evaluation: events are sorted by
// version (introduced "0" first; at equal versions introduced precedes
// fixed/last_affected) and applied in order.
func inRange(events []Event, version string, cmp Comparer) (bool, error) {
	if _, err := cmp.Compare(version, version); err != nil {
		return false, err
	}
	var evs []sortedEvent
	for _, e := range events {
		switch {
		case e.Introduced != "":
			evs = append(evs, sortedEvent{'i', e.Introduced})
		case e.Fixed != "":
			evs = append(evs, sortedEvent{'f', e.Fixed})
		case e.LastAffected != "":
			evs = append(evs, sortedEvent{'l', e.LastAffected})
		}
	}
	isZeroEv := func(e sortedEvent) bool { return e.kind == 'i' && e.version == "0" }
	// Validate versions up front so sorting cannot fail midway.
	for _, e := range evs {
		if isZeroEv(e) {
			continue
		}
		if _, err := cmp.Compare(e.version, e.version); err != nil {
			return false, err
		}
	}
	isZero := func(e sortedEvent) bool { return e.kind == 'i' && e.version == "0" }
	less := func(a, b sortedEvent) bool {
		if isZero(a) || isZero(b) {
			return isZero(a) && !isZero(b)
		}
		c, _ := cmp.Compare(a.version, b.version)
		if c != 0 {
			return c < 0
		}
		return a.kind == 'i' && b.kind != 'i'
	}
	sort.SliceStable(evs, func(i, j int) bool { return less(evs[i], evs[j]) })

	affected := false
	for _, e := range evs {
		zero := isZero(e)
		var c int
		if !zero {
			c, _ = cmp.Compare(version, e.version)
		}
		switch e.kind {
		case 'i':
			if zero || c >= 0 {
				affected = true
			}
		case 'f':
			if c >= 0 {
				affected = false
			}
		case 'l':
			if c > 0 {
				affected = false
			}
		}
	}
	return affected, nil
}

func buildFinding(p Package, acc *matchAcc, cmp Comparer) Finding {
	rec := acc.rec
	f := Finding{
		Package: p,
		VulnID:  rec.ID,
		Summary: rec.Summary,
	}
	if f.Summary == "" {
		f.Summary = firstLine(rec.Details)
	}
	f.Aliases = append([]string(nil), rec.Aliases...)
	sort.Strings(f.Aliases)
	seen := map[string]bool{}
	for _, id := range append([]string{rec.ID}, f.Aliases...) {
		if strings.HasPrefix(id, "CVE-") && !seen[id] {
			seen[id] = true
			f.CVEs = append(f.CVEs, id)
		}
	}
	sort.Strings(f.CVEs)

	sev := append([]Severity(nil), rec.Severity...)
	for _, a := range acc.affected {
		sev = append(sev, a.Severity...)
	}
	fillSeverity(&f, sev, rec.DatabaseSpecific, acc.affected)

	// Fixed versions: dedupe, drop unparseable, sort.
	uniq := map[string]bool{}
	for _, v := range acc.fixed {
		if _, err := cmp.Compare(v, v); err != nil || uniq[v] {
			continue
		}
		uniq[v] = true
		f.FixedVersions = append(f.FixedVersions, v)
	}
	sort.SliceStable(f.FixedVersions, func(i, j int) bool {
		c, _ := cmp.Compare(f.FixedVersions[i], f.FixedVersions[j])
		return c < 0
	})
	f.MinFixedVersion, f.FixCrossLine = minFixed(f.FixedVersions, p.Version, cmp)
	return f
}

func minFixed(sorted []string, installed string, cmp Comparer) (string, bool) {
	line, lerr := cmp.Line(installed)
	var firstAbove string
	for _, v := range sorted {
		if c, err := cmp.Compare(v, installed); err != nil || c <= 0 {
			continue
		}
		if firstAbove == "" {
			firstAbove = v
		}
		if l, err := cmp.Line(v); lerr == nil && err == nil && l == line {
			return v, false
		}
	}
	return firstAbove, firstAbove != ""
}

func fillSeverity(f *Finding, sev []Severity, dbSpecific map[string]any, affected []*Affected) {
	pick := func(typ string) string {
		for _, s := range sev {
			if s.Type == typ && s.Score != "" {
				return s.Score
			}
		}
		return ""
	}
	if v := pick("CVSS_V3"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.CVSSScore, f.HasScore = n, true
		} else {
			f.CVSSVector = v
			if score, err := CVSSV3BaseScore(v); err == nil {
				f.CVSSScore, f.HasScore = score, true
			}
		}
	} else if v := pick("CVSS_V4"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.CVSSScore, f.HasScore = n, true
		} else {
			f.CVSSVector = v // v4 base score is not computed
		}
	}
	if s, ok := dbSpecific["severity"].(string); ok && s != "" {
		f.Severity = strings.ToUpper(s)
	} else {
		for _, a := range affected {
			if s, ok := a.DatabaseSpecific["severity"].(string); ok && s != "" {
				f.Severity = strings.ToUpper(s)
				break
			}
		}
	}
	if f.Severity == "" && f.HasScore {
		f.Severity = SeverityLabel(f.CVSSScore)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// String implements fmt.Stringer for log output.
func (f Finding) String() string {
	return fmt.Sprintf("%s %s@%s fixed=%s", f.VulnID, f.Package.Name, f.Package.Version, f.MinFixedVersion)
}
