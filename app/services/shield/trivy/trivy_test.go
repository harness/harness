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

package trivy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/harness/gitness/app/services/shield/sast"
	"github.com/harness/gitness/types/enum"
)

func load(t *testing.T, name string) *sast.Report {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rep, err := sast.ParseSARIF(f)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func TestImageFindings(t *testing.T) {
	fs := ParseImageFindings(load(t, "image.sarif"))
	if len(fs) != 3 {
		t.Fatalf("want 3 findings, got %d", len(fs))
	}

	osf := fs[0]
	if osf.VulnID != "CVE-2023-5678" || osf.Package != "libssl3" || osf.InstalledVersion != "3.1.3-r0" ||
		!reflect.DeepEqual(osf.FixedVersions, []string{"3.1.4-r0"}) {
		t.Errorf("os finding: %+v", osf)
	}
	if osf.Ecosystem != EcosystemOSPackage || osf.Severity != sast.SeverityMedium || osf.Score != 6.5 {
		t.Errorf("os eco/sev: %s %s %v", osf.Ecosystem, osf.Severity, osf.Score)
	}
	if osf.Title != "openssl: Excessive time spent in DH check" || osf.Target != "library/alpine" {
		t.Errorf("os title/target: %q %q", osf.Title, osf.Target)
	}

	lib := fs[1]
	if lib.Ecosystem != EcosystemLibrary || lib.Package != "qs" || lib.Target != "app/package-lock.json" {
		t.Errorf("lib finding: %+v", lib)
	}
	if !reflect.DeepEqual(lib.FixedVersions, []string{"6.10.3", "6.9.7", "6.8.3"}) || lib.Severity != sast.SeverityHigh {
		t.Errorf("lib fixed/sev: %v %s", lib.FixedVersions, lib.Severity)
	}

	unfixed := fs[2]
	if len(unfixed.FixedVersions) != 0 || unfixed.Severity != sast.SeverityCritical || unfixed.Score != 9.5 {
		t.Errorf("unfixed: %+v", unfixed)
	}
	if osf.Fingerprint == lib.Fingerprint || osf.Fingerprint == unfixed.Fingerprint {
		t.Error("fingerprints must differ")
	}
}

func TestToShieldImageFinding(t *testing.T) {
	fs := ParseImageFindings(load(t, "image.sarif"))
	sf := ToShieldImageFinding(fs[1], 7, 9, 1000)
	if sf.Kind != enum.ShieldFindingKindImage || sf.Scanner != enum.ShieldScannerTrivy ||
		sf.Ecosystem != EcosystemLibrary || sf.Package != "qs" || sf.InstalledVersion != "6.5.2" ||
		sf.VulnID != "CVE-2022-24999" || sf.MinFixedVersion != "6.10.3" || sf.RiskScore != 75 ||
		sf.State != enum.ShieldFindingStateOpen || sf.RepoID != 7 || sf.LastSeenScanID != 9 {
		t.Errorf("unexpected: %+v", sf)
	}
	if sf.Aliases == nil || len(sf.FixedVersions) != 3 {
		t.Errorf("slices: %v %v", sf.Aliases, sf.FixedVersions)
	}
	got := ToShieldImageFinding(fs[2], 1, 1, 1)
	if got.FixedVersions == nil || len(got.FixedVersions) != 0 || got.MinFixedVersion != "" {
		t.Errorf("unfixed: %+v", got)
	}
}

func TestImageCVEAlias(t *testing.T) {
	rep := &sast.Report{Runs: []sast.Run{{Results: []sast.Result{{
		RuleID: "GHSA-xxxx-yyyy-zzzz",
		Message: sast.Text{
			Text: "Package: foo\nInstalled Version: 1\nVulnerability CVE-2024-1\nSeverity: LOW\nFixed Version: 2",
		},
	}}}}}
	fs := ParseImageFindings(rep)
	if len(fs) != 1 || fs[0].CVE != "CVE-2024-1" {
		t.Fatalf("%+v", fs)
	}
	if sf := ToShieldImageFinding(fs[0], 1, 1, 1); !reflect.DeepEqual(sf.Aliases, []string{"CVE-2024-1"}) {
		t.Errorf("aliases: %v", sf.Aliases)
	}
}

func TestConfigFindings(t *testing.T) {
	fs := ParseConfigFindings(load(t, "config.sarif"))
	if len(fs) != 3 {
		t.Fatalf("want 3 findings (rule-less skipped), got %d", len(fs))
	}
	a := fs[0]
	if a.RuleID != "AVD-AWS-0086" || a.Path != "infra/s3.tf" || a.StartLine != 12 || a.EndLine != 18 ||
		a.Severity != sast.SeverityHigh || a.Score != 8.0 {
		t.Errorf("a: %+v", a)
	}
	if a.Message != "No public access block so not blocking public acls" ||
		a.Title != "S3 Access block should block public ACL" {
		t.Errorf("a text: %q / %q", a.Message, a.Title)
	}
	c := fs[2]
	if c.Path != "Dockerfile" || c.EndLine != 3 || c.Severity != sast.SeverityLow || c.Score != 2.0 {
		t.Errorf("c: %+v", c)
	}

	sf := ToShieldConfigFinding(a, 3, 4, 50)
	if sf.Kind != enum.ShieldFindingKindIaC || sf.Scanner != enum.ShieldScannerTrivy ||
		sf.VulnID != "AVD-AWS-0086" || sf.ManifestPath != "infra/s3.tf" || sf.RiskScore != 80 ||
		sf.Package != "" || sf.Summary != a.Message {
		t.Errorf("unexpected: %+v", sf)
	}
}

func TestEcosystemOf(t *testing.T) {
	for target, want := range map[string]string{
		"library/alpine":             EcosystemOSPackage,
		"alpine:3.18 (alpine 3.18)":  EcosystemOSPackage,
		"app/package-lock.json":      EcosystemLibrary,
		"go.mod":                     EcosystemLibrary,
		"usr/lib/app.jar":            EcosystemLibrary,
		"opt/venv/lib/x/METADATA":    EcosystemLibrary,
		"srv/Gemfile.lock":           EcosystemLibrary,
		"srv/Gemfile":                EcosystemLibrary,
		"home/u/requirements.txt":    EcosystemLibrary,
		"registry.local/team/app:v1": EcosystemOSPackage,
	} {
		if got := ecosystemOf(target); got != want {
			t.Errorf("%s: got %s want %s", target, got, want)
		}
	}
}

func TestArgs(t *testing.T) {
	got := ImageArgs("alpine:3.18", Options{})
	want := []string{"image", "--format", "sarif", "--quiet", "--scanners", "vuln", "--", "alpine:3.18"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("image args: %v", got)
	}
	got = ImageArgs("r/x:1", Options{SkipDBUpdate: true, CacheDir: "/c"})
	want = []string{"image", "--format", "sarif", "--quiet", "--scanners", "vuln",
		"--skip-db-update", "--cache-dir", "/c", "--", "r/x:1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("offline image args: %v", got)
	}
	got = ConfigArgs("/w", Options{})
	want = []string{"config", "--format", "sarif", "--quiet", "--", "/w"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("config args: %v", got)
	}
	got = ConfigArgs("/w", Options{SkipCheckUpdate: true, CacheDir: "/c"})
	want = []string{"config", "--format", "sarif", "--quiet", "--skip-check-update", "--cache-dir", "/c", "--", "/w"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("offline config args: %v", got)
	}
}

func TestCommandConstruction(t *testing.T) {
	r := &ExecRunner{LookPath: func(s string) (string, error) { return "/opt/bin/" + s, nil }}
	cmd, err := r.ImageCommand(context.Background(), "alpine:3", Options{Binary: "mytrivy"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != "/opt/bin/mytrivy" || cmd.Args[1] != "image" || cmd.Args[len(cmd.Args)-1] != "alpine:3" {
		t.Errorf("image cmd: %v", cmd.Args)
	}
	cmd, err = r.ConfigCommand(context.Background(), "/work", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != "/opt/bin/trivy" || cmd.Dir != "/work" || cmd.Args[1] != "config" {
		t.Errorf("config cmd: %v dir=%s", cmd.Args, cmd.Dir)
	}
}

func TestMissingBinary(t *testing.T) {
	r := &ExecRunner{LookPath: func(string) (string, error) { return "", exec.ErrNotFound }}
	if _, err := r.ScanImage(context.Background(), "alpine:3", Options{}); !errors.Is(err, ErrBinaryMissing) {
		t.Errorf("image: want ErrBinaryMissing, got %v", err)
	}
	if _, err := r.ScanConfig(context.Background(), "/w", Options{}); !errors.Is(err, ErrBinaryMissing) {
		t.Errorf("config: want ErrBinaryMissing, got %v", err)
	}
}
