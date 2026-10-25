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

package sast

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/harness/gitness/types/enum"
)

func load(t *testing.T) []Finding {
	t.Helper()
	f, err := os.Open("testdata/semgrep.sarif")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rep, err := ParseSARIF(f)
	if err != nil {
		t.Fatal(err)
	}
	return rep.Findings()
}

func TestParseAndMap(t *testing.T) {
	fs := load(t)
	if len(fs) != 3 {
		t.Fatalf("want 3 findings (rule-less skipped), got %d", len(fs))
	}

	a := fs[0]
	if a.Severity != SeverityCritical || a.Score != 9.1 {
		t.Errorf("a severity: %s %v", a.Severity, a.Score)
	}
	if !reflect.DeepEqual(a.CWEs, []string{"CWE-89"}) {
		t.Errorf("a cwes: %v", a.CWEs)
	}
	if a.Path != "app/db.py" || a.StartLine != 42 || a.EndLine != 42 {
		t.Errorf("a location: %s %d-%d", a.Path, a.StartLine, a.EndLine)
	}
	if !strings.HasPrefix(a.Title, "Detected user input") || a.Tool != "Semgrep OSS" {
		t.Errorf("a title/tool: %q %q", a.Title, a.Tool)
	}

	b := fs[1]
	if b.Severity != SeverityMedium || b.Score != 5.3 {
		t.Errorf("b severity: %s %v", b.Severity, b.Score)
	}
	if !reflect.DeepEqual(b.CWEs, []string{"CWE-327", "CWE-328"}) {
		t.Errorf("b cwes: %v", b.CWEs)
	}
	if b.Path != "src/util/hash.go" {
		t.Errorf("b path: %q", b.Path)
	}
	if !strings.HasPrefix(b.Fingerprint, "generic.secrets.weak-hash|src/util/hash.go|") {
		t.Errorf("b fingerprint: %q", b.Fingerprint)
	}

	c := fs[2]
	if c.Severity != SeverityLow || c.Score != 2.0 || c.Title != "Leftover TODO" {
		t.Errorf("c: %+v", c)
	}
}

func TestSeverityThresholds(t *testing.T) {
	cases := []struct {
		s    float64
		want Severity
	}{{9, SeverityCritical}, {8.9, SeverityHigh}, {7, SeverityHigh},
		{6.9, SeverityMedium}, {4, SeverityMedium}, {3.9, SeverityLow}}
	for _, c := range cases {
		s := c.s
		if got, _ := severityAndScore(&s, "error"); got != c.want {
			t.Errorf("%v: got %s want %s", c.s, got, c.want)
		}
	}
	levels := map[string]Severity{"error": SeverityHigh, "warning": SeverityMedium, "note": SeverityLow, "": SeverityLow}
	for level, want := range levels {
		if got, _ := severityAndScore(nil, level); got != want {
			t.Errorf("level %q: got %s want %s", level, got, want)
		}
	}
}

func TestFingerprintIgnoresLinesAndWhitespace(t *testing.T) {
	mk := func(line int, snippet string) Finding {
		res := &Result{RuleID: "r", Message: Text{Text: "m"}, Locations: []Location{{PhysicalLocation: PhysicalLocation{
			ArtifactLocation: ArtifactLocation{URI: "a.go"},
			Region:           Region{StartLine: line, Snippet: Text{Text: snippet}},
		}}}}
		f, _ := buildFinding(Driver{}, nil, res)
		return f
	}
	if mk(1, "x  :=  1").Fingerprint != mk(99, "x := 1\n").Fingerprint {
		t.Error("fingerprint must not depend on line or whitespace")
	}
	if mk(1, "x := 1").Fingerprint == mk(1, "x := 2").Fingerprint {
		t.Error("different code must differ")
	}
}

func TestLenientParsing(t *testing.T) {
	rep, err := ParseSARIF(strings.NewReader(`{"runs":[{"results":[{"ruleId":"x","properties":"bogus","locations":[{}]}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	fs := rep.Findings()
	if len(fs) != 1 || fs[0].Title != "x" || fs[0].Severity != SeverityLow {
		t.Errorf("unexpected: %+v", fs)
	}
	if _, err := ParseSARIF(strings.NewReader("not json")); err == nil {
		t.Error("want error")
	}
}

func TestToShieldFinding(t *testing.T) {
	f := load(t)[0]
	sf := ToShieldFinding(f, 7, 3, 1000)
	if sf.Kind != enum.ShieldFindingKindSAST || sf.RepoID != 7 || sf.VulnID != f.RuleID ||
		sf.RiskScore != 91 || sf.CVSSScore != 9.1 || sf.ManifestPath != "app/db.py" ||
		sf.State != enum.ShieldFindingStateOpen || sf.Fingerprint != f.Fingerprint {
		t.Errorf("unexpected: %+v", sf)
	}
}

func TestCommandConstruction(t *testing.T) {
	r := &ExecRunner{LookPath: func(s string) (string, error) { return "/usr/bin/" + s, nil }}
	cmd, err := r.Command(context.Background(), "/repo", Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/usr/bin/opengrep", "scan", "--sarif", "--config", "p/default", "--metrics=off", "--quiet", "/repo"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("got %v", cmd.Args)
	}
	cmd, _ = r.Command(context.Background(), "/repo", Options{Binary: "semgrep", Config: "/rules"})
	if cmd.Args[0] != "/usr/bin/semgrep" || cmd.Args[4] != "/rules" {
		t.Errorf("got %v", cmd.Args)
	}
}

func TestMissingBinary(t *testing.T) {
	r := &ExecRunner{LookPath: func(string) (string, error) { return "", exec.ErrNotFound }}
	_, err := r.Run(context.Background(), t.TempDir(), Options{Binary: "nope"})
	if !errors.Is(err, ErrBinaryMissing) {
		t.Errorf("got %v", err)
	}
}
