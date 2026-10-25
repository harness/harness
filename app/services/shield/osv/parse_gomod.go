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
	"fmt"
	"io"
	"strings"
)

// GoModParser reads go.mod files. Every module in a require directive is
// reported (direct and "// indirect"). replace/exclude directives are not
// applied, and the go/toolchain directives are not reported as "stdlib".
type GoModParser struct{}

func (GoModParser) Ecosystem() string { return EcosystemGo }

func (GoModParser) Handles(filePath string) bool { return baseName(filePath) == "go.mod" }

func (GoModParser) Parse(filePath string, r io.Reader) (*ParseResult, error) {
	var pkgs []Package
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	inRequire := false
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		code, comment, _ := strings.Cut(line, "//")
		code = strings.TrimSpace(code)
		indirect := strings.Contains(comment, "indirect")
		if code == "" {
			continue
		}
		switch {
		case inRequire:
			if code == ")" {
				inRequire = false
				continue
			}
			p, err := goRequireLine(code, filePath, indirect)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			pkgs = append(pkgs, p)
		case code == "require (" || code == "require(":
			inRequire = true
		case strings.HasPrefix(code, "require ") || strings.HasPrefix(code, "require\t"):
			p, err := goRequireLine(strings.TrimSpace(code[len("require"):]), filePath, indirect)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			pkgs = append(pkgs, p)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return &ParseResult{Packages: dedupePackages(pkgs)}, nil
}

func goRequireLine(code, filePath string, indirect bool) (Package, error) {
	f := strings.Fields(code)
	if len(f) != 2 {
		return Package{}, fmt.Errorf("malformed require %q", code)
	}
	return Package{
		Ecosystem: EcosystemGo,
		Name:      strings.Trim(f[0], `"`),
		Version:   strings.Trim(f[1], `"`),
		Path:      filePath,
		Direct:    !indirect,
	}, nil
}
