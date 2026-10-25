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
	"bufio"
	"io"
	"path"
	"regexp"
	"strings"
)

// PyPIRequirementsParser reads requirements*.txt files. Only exact pins
// ("name==1.2.3", optionally with extras, environment markers and --hash
// options) produce packages. Ranges (>=, ~=, <), wildcard pins (==1.*),
// arbitrary equality (===), unversioned names, URLs and VCS requirements are
// reported in ParseResult.Unpinned. Option lines (-r, -c, -e, --index-url...)
// are skipped; included files are not followed. All packages are Direct.
type PyPIRequirementsParser struct{}

func (PyPIRequirementsParser) Ecosystem() string { return EcosystemPyPI }

func (PyPIRequirementsParser) Handles(filePath string) bool {
	b := strings.ToLower(baseName(filePath))
	if strings.HasPrefix(b, "requirements") && strings.HasSuffix(b, ".txt") {
		return true
	}
	// requirements/base.txt, requirements/dev.txt layout
	return path.Ext(b) == ".txt" && strings.EqualFold(path.Base(path.Dir(filePath)), "requirements")
}

var (
	pyReqRe  = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*(\[[^\]]*\])?\s*(.*)$`)
	pyPinRe  = regexp.MustCompile(`^==\s*([A-Za-z0-9][A-Za-z0-9.!+_*-]*)$`)
	pyNormRe = regexp.MustCompile(`[-_.]+`)
)

// NormalizePyPIName applies PEP 503 name normalisation, which is the form
// used by the OSV PyPI ecosystem.
func NormalizePyPIName(n string) string {
	return strings.ToLower(pyNormRe.ReplaceAllString(n, "-"))
}

func (PyPIRequirementsParser) Parse(filePath string, r io.Reader) (*ParseResult, error) {
	res := &ParseResult{}
	var pkgs []Package
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var (
		buf       strings.Builder
		startLn   int
		lineNo    int
		continued bool
	)
	flush := func() {
		text := strings.TrimSpace(buf.String())
		buf.Reset()
		if text == "" {
			return
		}
		pkg, unp := parsePyRequirement(text, filePath, startLn)
		if pkg != nil {
			pkgs = append(pkgs, *pkg)
		}
		if unp != nil {
			res.Unpinned = append(res.Unpinned, *unp)
		}
	}
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if !continued {
			startLn = lineNo
		}
		if i := commentIndex(line); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimRight(line, " \t\r")
		if strings.HasSuffix(line, "\\") {
			buf.WriteString(strings.TrimSuffix(line, "\\"))
			buf.WriteByte(' ')
			continued = true
			continue
		}
		buf.WriteString(line)
		continued = false
		flush()
	}
	if continued {
		flush()
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	res.Packages = dedupePackages(pkgs)
	return res, nil
}

// commentIndex finds a '#' at line start or preceded by whitespace.
func commentIndex(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
			return i
		}
	}
	return -1
}

func parsePyRequirement(text, filePath string, line int) (*Package, *Unpinned) {
	if strings.HasPrefix(text, "-") {
		return nil, nil // pip option (-r, -c, -e, --index-url, ...)
	}
	unp := func(name, reason string) (*Package, *Unpinned) {
		return nil, &Unpinned{
			Ecosystem: EcosystemPyPI, Name: name, Spec: text, Path: filePath, Line: line, Reason: reason,
		}
	}
	if strings.Contains(text, "://") || strings.HasPrefix(text, "git+") ||
		strings.HasPrefix(text, ".") || strings.HasPrefix(text, "/") {
		return unp("", "url, vcs or path requirement")
	}
	spec := text
	if i := strings.Index(spec, ";"); i >= 0 {
		spec = spec[:i]
	}
	if i := strings.Index(spec, " --"); i >= 0 {
		spec = spec[:i]
	}
	m := pyReqRe.FindStringSubmatch(strings.TrimSpace(spec))
	if m == nil {
		return unp("", "unparseable requirement")
	}
	name, constraint := m[1], strings.TrimSpace(m[3])
	if constraint == "" {
		return unp(name, "no version specified")
	}
	pin := pyPinRe.FindStringSubmatch(constraint)
	if pin == nil {
		return unp(name, "not an exact == pin")
	}
	if strings.Contains(pin[1], "*") {
		return unp(name, "wildcard pin")
	}
	return &Package{
		Ecosystem: EcosystemPyPI,
		Name:      NormalizePyPIName(name),
		Version:   pin[1],
		Path:      filePath,
		Direct:    true,
	}, nil
}
