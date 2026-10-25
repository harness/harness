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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func parseFixture(t *testing.T, name string) (*ParseResult, error) {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "lock", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	return NewRegistry().Parse(name, f)
}

func TestParseGoMod(t *testing.T) {
	res, err := parseFixture(t, "go.mod")
	if err != nil {
		t.Fatal(err)
	}
	want := []Package{
		{EcosystemGo, "github.com/foo/bar", "v2.0.0+incompatible", "go.mod", true},
		{EcosystemGo, "github.com/gin-gonic/gin", "v1.7.0", "go.mod", true},
		{EcosystemGo, "github.com/pkg/errors", "v0.9.1", "go.mod", false},
		{EcosystemGo, "golang.org/x/crypto", "v0.0.0-20210921155107-089bfa567519", "go.mod", false},
		{EcosystemGo, "golang.org/x/text", "v0.3.6", "go.mod", true},
	}
	if !reflect.DeepEqual(res.Packages, want) {
		t.Fatalf("got %+v\nwant %+v", res.Packages, want)
	}
}

func TestParseGoModErrors(t *testing.T) {
	_, err := NewRegistry().Parse("go.mod", strings.NewReader("require (\n\tonly-name\n)\n"))
	if err == nil {
		t.Fatal("expected error for malformed require")
	}
}

func TestParseNPM(t *testing.T) {
	res, err := parseFixture(t, "package-lock.json")
	if err != nil {
		t.Fatal(err)
	}
	const p = "package-lock.json"
	want := []Package{
		{EcosystemNPM, "@scope/pkg", "1.2.3", p, true},
		{EcosystemNPM, "lodash", "4.17.15", p, true},
		{EcosystemNPM, "minimist", "0.0.8", p, false},
		{EcosystemNPM, "minimist", "1.2.0", p, true},
	}
	if !reflect.DeepEqual(res.Packages, want) {
		t.Fatalf("got %+v\nwant %+v", res.Packages, want)
	}
}

func TestParseNPMUnsupported(t *testing.T) {
	_, err := parseFixture(t, "package-lock-v1.json")
	// The fixture is named differently, so go through the parser directly.
	if !errors.Is(err, ErrNoParser) {
		t.Fatalf("expected ErrNoParser, got %v", err)
	}
	f, _ := os.Open(filepath.Join("testdata", "lock", "package-lock-v1.json"))
	defer f.Close()
	_, err = NPMLockParser{}.Parse("package-lock.json", f)
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestParseNPMAlias(t *testing.T) {
	in := `{"lockfileVersion":2,"packages":{"":{"dependencies":{"mylodash":"npm:lodash@4"}},
	"node_modules/mylodash":{"name":"lodash","version":"4.17.20"}}}`
	res, err := NPMLockParser{}.Parse("package-lock.json", strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Packages) != 1 || res.Packages[0].Name != "lodash" || !res.Packages[0].Direct {
		t.Fatalf("got %+v", res.Packages)
	}
}

func TestParsePyPI(t *testing.T) {
	res, err := parseFixture(t, "requirements.txt")
	if err != nil {
		t.Fatal(err)
	}
	const p = "requirements.txt"
	want := []Package{
		{EcosystemPyPI, "django", "3.2.0", p, true},
		{EcosystemPyPI, "flask-login", "0.4.1", p, true},
		{EcosystemPyPI, "requests", "2.19.0", p, true},
	}
	if !reflect.DeepEqual(res.Packages, want) {
		t.Fatalf("got %+v\nwant %+v", res.Packages, want)
	}
	reasons := map[string]string{}
	for _, u := range res.Unpinned {
		reasons[u.Spec] = u.Reason
	}
	for spec, reason := range map[string]string{
		"numpy>=1.20":                          "not an exact == pin",
		"pyyaml":                               "no version specified",
		"urllib3~=1.26.0":                      "not an exact == pin",
		"six==1.*":                             "wildcard pin",
		"git+https://github.com/x/y.git#egg=y": "url, vcs or path requirement",
		"./local/pkg":                          "url, vcs or path requirement",
	} {
		if reasons[spec] != reason {
			t.Errorf("unpinned %q: got %q want %q (all: %v)", spec, reasons[spec], reason, reasons)
		}
	}
	if len(res.Unpinned) != 6 {
		t.Errorf("got %d unpinned, want 6: %+v", len(res.Unpinned), res.Unpinned)
	}
}

func TestRegistryHandles(t *testing.T) {
	r := NewRegistry()
	for path, want := range map[string]bool{
		"go.mod":                true,
		"svc/api/go.mod":        true,
		"go.sum":                false,
		"web/package-lock.json": true,
		"npm-shrinkwrap.json":   true,
		"package.json":          false,
		"requirements.txt":      true,
		"requirements-dev.txt":  true,
		"requirements/base.txt": true,
		"docs/notes.txt":        false,
		"README.md":             false,
	} {
		if got := r.Handles(path); got != want {
			t.Errorf("Handles(%q)=%v want %v", path, got, want)
		}
	}
	if _, err := r.Parse("Cargo.lock", strings.NewReader("")); !errors.Is(err, ErrNoParser) {
		t.Errorf("expected ErrNoParser, got %v", err)
	}
}

func TestNormalizePyPIName(t *testing.T) {
	for in, want := range map[string]string{
		"Django": "django", "Flask_Login": "flask-login", "zope.interface": "zope-interface", "A--B__c": "a-b-c",
	} {
		if got := NormalizePyPIName(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}
