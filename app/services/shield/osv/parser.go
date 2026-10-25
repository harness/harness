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

// Package osv contains lockfile parsers, an OSV vulnerability database
// loader/fetcher and a version-aware matcher. It has no database or API
// dependencies.
package osv

import (
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
)

// OSV ecosystem names used by the built-in parsers.
const (
	EcosystemGo   = "Go"
	EcosystemNPM  = "npm"
	EcosystemPyPI = "PyPI"
)

// ErrNoParser is returned when no registered parser handles a file.
var ErrNoParser = errors.New("osv: no parser for file")

// ErrUnsupportedFormat is returned by a parser that recognises a file
// but not the specific format version (e.g. npm lockfileVersion 1).
var ErrUnsupportedFormat = errors.New("osv: unsupported file format")

// Package is a single dependency found in a lockfile or manifest.
type Package struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	// Path is the file the package was read from, as given to the parser.
	Path string `json:"path"`
	// Direct is true when the manifest declares the package itself,
	// false for transitive (indirect) dependencies.
	Direct bool `json:"direct"`
}

// Unpinned describes a dependency declaration that could not be turned
// into an exact version, and was therefore not scanned.
type Unpinned struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name,omitempty"`
	Spec      string `json:"spec"`
	Path      string `json:"path"`
	Line      int    `json:"line,omitempty"`
	Reason    string `json:"reason"`
}

// ParseResult is the output of a Parser.
type ParseResult struct {
	Packages []Package  `json:"packages"`
	Unpinned []Unpinned `json:"unpinned,omitempty"`
}

// Parser extracts packages from one kind of lockfile or manifest.
// To support a new ecosystem implement Parser and Register it.
type Parser interface {
	// Ecosystem returns the OSV ecosystem name of the produced packages.
	Ecosystem() string
	// Handles reports whether the parser applies to the given file path.
	Handles(filePath string) bool
	// Parse reads the file content. filePath is recorded in Package.Path.
	Parse(filePath string, r io.Reader) (*ParseResult, error)
}

// Registry is a set of parsers.
type Registry struct {
	parsers []Parser
}

// NewRegistry returns a registry with the built-in parsers (Go, npm, PyPI).
func NewRegistry() *Registry {
	r := &Registry{}
	r.Register(GoModParser{})
	r.Register(NPMLockParser{})
	r.Register(PyPIRequirementsParser{})
	return r
}

// Register adds a parser. Earlier registered parsers win on overlap.
func (r *Registry) Register(p Parser) {
	r.parsers = append(r.parsers, p)
}

// Handles reports whether any parser handles the path.
func (r *Registry) Handles(filePath string) bool {
	return r.find(filePath) != nil
}

func (r *Registry) find(filePath string) Parser {
	for _, p := range r.parsers {
		if p.Handles(filePath) {
			return p
		}
	}
	return nil
}

// Parse selects the parser for filePath and runs it.
func (r *Registry) Parse(filePath string, rd io.Reader) (*ParseResult, error) {
	p := r.find(filePath)
	if p == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoParser, filePath)
	}
	res, err := p.Parse(filePath, rd)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}
	return res, nil
}

func baseName(p string) string { return path.Base(p) }

// dedupePackages merges packages with equal (ecosystem, name, version, path),
// OR-ing Direct, and returns them in a stable order.
func dedupePackages(in []Package) []Package {
	type key struct{ eco, name, ver, path string }
	idx := make(map[key]int, len(in))
	out := make([]Package, 0, len(in))
	for _, p := range in {
		k := key{p.Ecosystem, p.Name, p.Version, p.Path}
		if i, ok := idx[k]; ok {
			out[i].Direct = out[i].Direct || p.Direct
			continue
		}
		idx[k] = len(out)
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Version < out[j].Version
	})
	return out
}
