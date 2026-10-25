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
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixtureDB(t *testing.T) *DB {
	t.Helper()
	db, err := LoadDir(filepath.Join("testdata", "records"))
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func ev(kv ...string) []Event {
	var out []Event
	for i := 0; i < len(kv); i += 2 {
		var e Event
		switch kv[i] {
		case "i":
			e.Introduced = kv[i+1]
		case "f":
			e.Fixed = kv[i+1]
		case "l":
			e.LastAffected = kv[i+1]
		}
		out = append(out, e)
	}
	return out
}

func TestInRange(t *testing.T) {
	sv := SemverComparer{}
	tests := []struct {
		name    string
		events  []Event
		version string
		want    bool
	}{
		{"introduced 0 below fix", ev("i", "0", "f", "1.5.0"), "1.4.9", true},
		{"at fix", ev("i", "0", "f", "1.5.0"), "1.5.0", false},
		{"above fix", ev("i", "0", "f", "1.5.0"), "2.0.0", false},
		{"below introduced", ev("i", "1.2.0", "f", "1.5.0"), "1.1.9", false},
		{"at introduced", ev("i", "1.2.0", "f", "1.5.0"), "1.2.0", true},
		{"open ended", ev("i", "1.2.0"), "9.9.9", true},
		{"last_affected inclusive", ev("i", "0", "l", "1.5.0"), "1.5.0", true},
		{"after last_affected", ev("i", "0", "l", "1.5.0"), "1.5.1", false},
		{"two windows, in second", ev("i", "0", "f", "1.0.0", "i", "2.0.0", "f", "2.3.0"), "2.1.0", true},
		{"two windows, gap", ev("i", "0", "f", "1.0.0", "i", "2.0.0", "f", "2.3.0"), "1.5.0", false},
		{"unsorted events", ev("f", "2.3.0", "i", "2.0.0", "f", "1.0.0", "i", "0"), "0.5.0", true},
		{"unsorted events gap", ev("f", "2.3.0", "i", "2.0.0", "f", "1.0.0", "i", "0"), "1.5.0", false},
		{"introduced equals fixed", ev("i", "1.0.0", "f", "1.0.0"), "1.0.0", false},
		{"prerelease before fix", ev("i", "0", "f", "1.0.0"), "1.0.0-rc.1", true},
		{"pseudo-version fixed", ev("i", "0", "f", "0.0.0-20220314234659-1baeb1ce4c0b"), "v0.0.0-20210921155107-089bfa567519", true},
		{"limit ignored", []Event{{Introduced: "0"}, {Limit: "1.0.0"}}, "5.0.0", true},
	}
	for _, tc := range tests {
		got, err := inRange(tc.events, tc.version, sv)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
	if _, err := inRange(ev("i", "0", "f", "garbage"), "1.0.0", sv); err == nil {
		t.Error("expected error for unparseable event version")
	}
}

func TestMatchFixtures(t *testing.T) {
	db := fixtureDB(t)
	reg := NewRegistry()
	var pkgs []Package
	for _, name := range []string{"go.mod", "package-lock.json", "requirements.txt"} {
		data, err := os.ReadFile(filepath.Join("testdata", "lock", name))
		if err != nil {
			t.Fatal(err)
		}
		res, err := reg.Parse(name, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		pkgs = append(pkgs, res.Packages...)
	}
	res := Match(db, pkgs)
	if len(res.Skipped) != 0 {
		t.Fatalf("skipped: %+v", res.Skipped)
	}
	type row struct {
		pkg, ver, id, minFixed string
		cross                  bool
		fixed                  []string
		cves                   []string
		score                  float64
		hasScore               bool
		sev                    string
	}
	var got []row
	for _, f := range res.Findings {
		got = append(got, row{f.Package.Name, f.Package.Version, f.VulnID, f.MinFixedVersion, f.FixCrossLine,
			f.FixedVersions, f.CVEs, f.CVSSScore, f.HasScore, f.Severity})
	}
	want := []row{
		{"github.com/gin-gonic/gin", "v1.7.0", "GO-2022-0001", "", false, nil, []string{"CVE-2022-0001"}, 0, false, ""},
		{"golang.org/x/crypto", "v0.0.0-20210921155107-089bfa567519", "GO-2022-0001",
			"0.0.0-20220314234659-1baeb1ce4c0b", false, []string{"0.0.0-20220314234659-1baeb1ce4c0b"},
			[]string{"CVE-2022-0001"}, 0, false, ""},
		{"golang.org/x/text", "v0.3.6", "GO-2021-0113", "0.3.7", false, []string{"0.3.7"},
			[]string{"CVE-2021-38561"}, 0, false, ""},
		{"lodash", "4.17.15", "GHSA-c2qf-rxjj-qqgw", "4.17.21", false, []string{"4.17.21"},
			[]string{"CVE-2021-23337"}, 7.2, true, "HIGH"},
		{"lodash", "4.17.15", "GHSA-p6mc-m468-83gw", "4.17.19", false, []string{"4.17.19"},
			[]string{"CVE-2020-8203"}, 7.4, true, "HIGH"},
		{"minimist", "0.0.8", "GHSA-xvch-5gv4-984h", "0.2.4", false, []string{"0.2.4", "1.2.6"},
			[]string{"CVE-2021-44906"}, 9.8, true, "CRITICAL"},
		{"minimist", "1.2.0", "GHSA-xvch-5gv4-984h", "1.2.6", false, []string{"0.2.4", "1.2.6"},
			[]string{"CVE-2021-44906"}, 9.8, true, "CRITICAL"},
		{"requests", "2.19.0", "PYSEC-2018-28", "2.20.0", false, []string{"2.20.0"},
			[]string{"CVE-2018-18074"}, 7.5, true, "HIGH"},
	}
	if !reflect.DeepEqual(got, want) {
		for _, g := range got {
			t.Logf("got  %+v", g)
		}
		t.Fatalf("findings mismatch")
	}
	// Spot-check fields not in the table.
	for _, f := range res.Findings {
		switch f.VulnID {
		case "GO-2022-0001":
			if f.CVSSVector == "" || f.HasScore {
				t.Errorf("CVSS v4 should keep the vector without a computed score: %+v", f)
			}
		case "GO-2021-0113":
			if !reflect.DeepEqual(f.Aliases, []string{"CVE-2021-38561", "GHSA-ppp9-7jff-5vj2"}) {
				t.Errorf("aliases %v", f.Aliases)
			}
			if f.Summary == "" {
				t.Error("missing summary")
			}
		}
	}
}

func TestMatchEdgeCases(t *testing.T) {
	db := fixtureDB(t)
	tests := []struct {
		name string
		pkg  Package
		ids  []string
	}{
		{"withdrawn is ignored (lodash fixed)", Package{Ecosystem: EcosystemNPM, Name: "lodash", Version: "4.17.21"}, nil},
		{"unknown package", Package{Ecosystem: EcosystemNPM, Name: "left-pad", Version: "1.0.0"}, nil},
		{"wrong ecosystem", Package{Ecosystem: EcosystemGo, Name: "lodash", Version: "4.17.15"}, nil},
		{"explicit versions list", Package{Ecosystem: EcosystemPyPI, Name: "django", Version: "3.1.1"},
			[]string{"PYSEC-2021-9"}},
		{"pypi range, not listed version", Package{Ecosystem: EcosystemPyPI, Name: "django", Version: "2.2.19"},
			[]string{"PYSEC-2021-9"}},
		{"pypi at fix", Package{Ecosystem: EcosystemPyPI, Name: "django", Version: "2.2.20"}, nil},
		{"pypi other line fixed", Package{Ecosystem: EcosystemPyPI, Name: "django", Version: "3.2.0"}, nil},
		{"pypi name normalised lookup", Package{Ecosystem: EcosystemPyPI, Name: "Django", Version: "3.0.1"},
			[]string{"PYSEC-2021-9"}},
		{"git range ignored", Package{Ecosystem: EcosystemGo, Name: "github.com/foo/bar", Version: "v2.0.0+incompatible"}, nil},
		{"last_affected inclusive", Package{Ecosystem: EcosystemGo, Name: "github.com/gin-gonic/gin", Version: "v1.7.0"},
			[]string{"GO-2022-0001"}},
		{"after last_affected", Package{Ecosystem: EcosystemGo, Name: "github.com/gin-gonic/gin", Version: "v1.7.1"}, nil},
	}
	for _, tc := range tests {
		res := Match(db, []Package{tc.pkg})
		var ids []string
		for _, f := range res.Findings {
			ids = append(ids, f.VulnID)
		}
		if !reflect.DeepEqual(ids, tc.ids) {
			t.Errorf("%s: got %v want %v", tc.name, ids, tc.ids)
		}
	}
}

func TestMatchMinFixedLine(t *testing.T) {
	db := fixtureDB(t)
	// Django 3.0.1 is in the 3.0 line: the minimal fix is 3.0.14, not 2.2.20 or 3.1.8.
	res := Match(db, []Package{{Ecosystem: EcosystemPyPI, Name: "django", Version: "3.0.1"}})
	if len(res.Findings) != 1 {
		t.Fatalf("got %d findings", len(res.Findings))
	}
	f := res.Findings[0]
	if f.MinFixedVersion != "3.0.14" || f.FixCrossLine {
		t.Errorf("min fixed %q cross=%v", f.MinFixedVersion, f.FixCrossLine)
	}
	if want := []string{"2.2.20", "3.0.14", "3.1.8"}; !reflect.DeepEqual(f.FixedVersions, want) {
		t.Errorf("fixed %v want %v", f.FixedVersions, want)
	}
	if f.Summary != "Django 2.2 before 2.2.20, 3.0 before 3.0.14, and 3.1 before 3.1.8 allows directory traversal." {
		t.Errorf("summary from first line of details: %q", f.Summary)
	}

	// Cross-line: only fix is in a later major.
	db2 := NewDB()
	db2.Add(&Record{ID: "X-1", Affected: []Affected{{
		Package: AffectedPackage{Ecosystem: EcosystemNPM, Name: "foo"},
		Ranges:  []Range{{Type: RangeSemver, Events: ev("i", "0", "f", "2.0.0")}},
	}}})
	res = Match(db2, []Package{{Ecosystem: EcosystemNPM, Name: "foo", Version: "1.4.0"}})
	if f := res.Findings[0]; f.MinFixedVersion != "2.0.0" || !f.FixCrossLine {
		t.Errorf("cross-line: %+v", f)
	}
}

func TestMatchSkipsUnparseableVersion(t *testing.T) {
	db := fixtureDB(t)
	res := Match(db, []Package{{Ecosystem: EcosystemNPM, Name: "lodash", Version: "not-a-version"}})
	if len(res.Findings) != 0 || len(res.Skipped) != 1 {
		t.Fatalf("got %+v", res)
	}
}

func TestMatchUnsupportedEcosystemRange(t *testing.T) {
	// ECOSYSTEM ranges of ecosystems without a comparer are ignored, but
	// explicit versions still match.
	db := NewDB()
	db.Add(&Record{ID: "DSA-1", Affected: []Affected{{
		Package:  AffectedPackage{Ecosystem: "Debian:11", Name: "openssl"},
		Ranges:   []Range{{Type: RangeEcosystem, Events: ev("i", "0", "f", "1.1.1n-0")}},
		Versions: []string{"1.1.1k-1"},
	}}})
	r1 := Match(db, []Package{{Ecosystem: "Debian:11", Name: "openssl", Version: "1.1.1a-1"}})
	if len(r1.Findings) != 0 {
		t.Errorf("range should be ignored: %+v", r1.Findings)
	}
	// "1.1.1k-1" parses as semver prerelease; explicit version matches by string.
	r2 := Match(db, []Package{{Ecosystem: "Debian:11", Name: "openssl", Version: "1.1.1k-1"}})
	if len(r2.Findings) != 1 {
		t.Errorf("explicit version should match: %+v", r2)
	}
}

func TestMatchDedupesAcrossAffectedEntries(t *testing.T) {
	db := NewDB()
	db.Add(&Record{ID: "X-2", Affected: []Affected{
		{Package: AffectedPackage{Ecosystem: EcosystemNPM, Name: "foo"}, Versions: []string{"1.0.0"}},
		{Package: AffectedPackage{Ecosystem: EcosystemNPM, Name: "foo"},
			Ranges: []Range{{Type: RangeSemver, Events: ev("i", "0", "f", "1.1.0")}}},
	}})
	res := Match(db, []Package{{Ecosystem: EcosystemNPM, Name: "foo", Version: "1.0.0"}})
	if len(res.Findings) != 1 || res.Findings[0].MinFixedVersion != "1.1.0" {
		t.Fatalf("got %+v", res.Findings)
	}
}

func TestSeverityHandling(t *testing.T) {
	tests := []struct {
		name     string
		sev      []Severity
		dbs      map[string]any
		vector   string
		score    float64
		hasScore bool
		label    string
	}{
		{"v3 vector", []Severity{{"CVSS_V3", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}}, nil,
			"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", 9.8, true, "CRITICAL"},
		{"numeric score provided", []Severity{{"CVSS_V3", "5.3"}}, nil, "", 5.3, true, "MEDIUM"},
		{"v3 preferred over v4", []Severity{
			{"CVSS_V4", "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:N/SI:N/SA:N"},
			{"CVSS_V3", "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N"}}, nil,
			"CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N", 3.3, true, "LOW"},
		{"invalid v3 vector keeps vector only", []Severity{{"CVSS_V3", "CVSS:3.1/garbage"}}, nil,
			"CVSS:3.1/garbage", 0, false, ""},
		{"db label wins", nil, map[string]any{"severity": "moderate"}, "", 0, false, "MODERATE"},
		{"none", nil, nil, "", 0, false, ""},
	}
	for _, tc := range tests {
		db := NewDB()
		db.Add(&Record{ID: "S-1", Severity: tc.sev, DatabaseSpecific: tc.dbs, Affected: []Affected{{
			Package: AffectedPackage{Ecosystem: EcosystemNPM, Name: "foo"}, Versions: []string{"1.0.0"}}}})
		res := Match(db, []Package{{Ecosystem: EcosystemNPM, Name: "foo", Version: "1.0.0"}})
		f := res.Findings[0]
		if f.CVSSVector != tc.vector || f.CVSSScore != tc.score || f.HasScore != tc.hasScore || f.Severity != tc.label {
			t.Errorf("%s: got %+v", tc.name, f)
		}
	}
}

func TestCVSSV3BaseScore(t *testing.T) {
	tests := []struct {
		vector string
		want   float64
	}{
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", 9.8},
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H", 10.0},
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N", 6.1},
		{"CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", 7.8},
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N", 0},
		{"CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N", 7.5},
		{"CVSS:3.1/AV:P/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N", 1.6},
	}
	for _, tc := range tests {
		got, err := CVSSV3BaseScore(tc.vector)
		if err != nil || got != tc.want {
			t.Errorf("%s: got %v, %v want %v", tc.vector, got, err, tc.want)
		}
	}
	for _, bad := range []string{"", "CVSS:4.0/AV:N", "CVSS:3.1/AV:X/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:Z/C:H/I:H/A:H"} {
		if _, err := CVSSV3BaseScore(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestLoadZipAndFilter(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	entries, _ := os.ReadDir(filepath.Join("testdata", "records"))
	for _, e := range entries {
		data, _ := os.ReadFile(filepath.Join("testdata", "records", e.Name()))
		w, _ := zw.Create(e.Name())
		_, _ = w.Write(data)
	}
	w, _ := zw.Create("README.txt")
	_, _ = w.Write([]byte("ignored"))
	_ = zw.Close()

	db, err := LoadZip(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if db.Len() != len(entries) {
		t.Errorf("Len=%d want %d", db.Len(), len(entries))
	}
	if _, ok := db.Record("GO-2021-0113"); !ok {
		t.Error("record missing")
	}
	res := Match(db, []Package{{Ecosystem: EcosystemNPM, Name: "lodash", Version: "4.17.15"}})
	if len(res.Findings) != 2 {
		t.Errorf("findings %d want 2", len(res.Findings))
	}

	filtered, err := LoadZip(bytes.NewReader(buf.Bytes()), int64(buf.Len()),
		WithPackageFilter(func(eco, name string) bool { return eco == EcosystemGo }))
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Len() != 2 { // GO-2021-0113 and GO-2022-0001
		t.Errorf("filtered Len=%d want 2", filtered.Len())
	}

	// A zip with a corrupt record fails and names the entry.
	var bad bytes.Buffer
	bz := zip.NewWriter(&bad)
	bw, _ := bz.Create("broken.json")
	_, _ = bw.Write([]byte("{not json"))
	_ = bz.Close()
	if _, err := LoadZip(bytes.NewReader(bad.Bytes()), int64(bad.Len())); err == nil || !strings.Contains(err.Error(), "broken.json") {
		t.Errorf("expected error naming broken.json, got %v", err)
	}
	if _, err := LoadZip(bytes.NewReader([]byte("nope")), 4); err == nil {
		t.Error("expected error for non-zip")
	}
}

func TestDBReplaceRecord(t *testing.T) {
	db := NewDB()
	mk := func(fixed string) *Record {
		return &Record{ID: "R-1", Affected: []Affected{{
			Package: AffectedPackage{Ecosystem: EcosystemNPM, Name: "foo"},
			Ranges:  []Range{{Type: RangeSemver, Events: ev("i", "0", "f", fixed)}}}}}
	}
	db.Add(mk("1.0.0"))
	db.Add(mk("2.0.0"))
	res := Match(db, []Package{{Ecosystem: EcosystemNPM, Name: "foo", Version: "1.5.0"}})
	if db.Len() != 1 || len(res.Findings) != 1 {
		t.Fatalf("len=%d findings=%d", db.Len(), len(res.Findings))
	}
}

func TestLoadDirErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDir(dir); err == nil {
		t.Error("expected error for record without id")
	}
}

// TestRealGoDatabase downloads the real Go ecosystem archive. Skipped unless
// OSV_REAL_DB=1 is set because it needs network access and ~10MB.
func TestRealGoDatabase(t *testing.T) {
	if os.Getenv("OSV_REAL_DB") != "1" {
		t.Skip("set OSV_REAL_DB=1 to run")
	}
	dest := filepath.Join(t.TempDir(), "go.zip")
	f := &Fetcher{}
	if _, err := f.FetchToFile(t.Context(), EcosystemGo, Validators{}, dest); err != nil {
		t.Fatal(err)
	}
	db, err := LoadZipFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	res := Match(db, []Package{{Ecosystem: EcosystemGo, Name: "golang.org/x/text", Version: "v0.3.6"}})
	found := false
	for _, fi := range res.Findings {
		if fi.VulnID == "GO-2021-0113" {
			found = true
		}
	}
	if !found {
		t.Errorf("GO-2021-0113 not reported for x/text v0.3.6 (db has %d records)", db.Len())
	}
}
