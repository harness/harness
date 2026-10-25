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

package sourcemap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrInvalidMap is returned for malformed source maps.
	ErrInvalidMap = errors.New("sourcemap: invalid source map")
	// ErrNotFound is returned by providers when no map exists for a file.
	ErrNotFound = errors.New("sourcemap: not found")
)

// MaxMapSize bounds the accepted input size of Parse.
const MaxMapSize = 64 << 20

// Mapping is a resolved original location.
type Mapping struct {
	Source string // original source path (sourceRoot applied)
	Line   int    // 1-based
	Column int    // 0-based
	Name   string // original identifier, may be empty
}

type segment struct {
	genCol  int32
	src     int32 // -1 when the segment has no source
	srcLine int32 // 0-based
	srcCol  int32
	name    int32 // -1 when none
}

type section struct {
	line, col int // 0-based offset
	m         *Map
}

// Map is a parsed source map. It is immutable and safe for concurrent use.
type Map struct {
	File           string
	Sources        []string // with sourceRoot applied
	SourcesContent []*string
	Names          []string

	lines    [][]segment // by 0-based generated line
	sections []section   // index map only
	size     int
}

type rawMap struct {
	Version        int               `json:"version"`
	File           string            `json:"file"`
	SourceRoot     string            `json:"sourceRoot"`
	Sources        []string          `json:"sources"`
	SourcesContent []*string         `json:"sourcesContent"`
	Names          []string          `json:"names"`
	Mappings       string            `json:"mappings"`
	Sections       []json.RawMessage `json:"sections"`
}

type rawSection struct {
	Offset struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"offset"`
	Map *json.RawMessage `json:"map"`
	URL string           `json:"url"`
}

// Parse parses a Source Map v3 document or an index map.
func Parse(data []byte) (*Map, error) {
	if len(data) > MaxMapSize {
		return nil, fmt.Errorf("%w: too large", ErrInvalidMap)
	}
	// Strip the XSSI guard some servers prepend.
	if bytes.HasPrefix(data, []byte(")]}'")) {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		}
	}
	return parse(data, 0)
}

//nolint:gocognit
func parse(data []byte, depth int) (*Map, error) {
	var r rawMap
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidMap, err)
	}
	if r.Version != 3 {
		return nil, fmt.Errorf("%w: unsupported version %d", ErrInvalidMap, r.Version)
	}
	m := &Map{File: r.File, Names: r.Names, SourcesContent: r.SourcesContent}

	if r.Sections != nil {
		if depth > 0 {
			return nil, fmt.Errorf("%w: nested index maps", ErrInvalidMap)
		}
		for i, rs := range r.Sections {
			var s rawSection
			if err := json.Unmarshal(rs, &s); err != nil {
				return nil, fmt.Errorf("%w: section %d: %w", ErrInvalidMap, i, err)
			}
			if s.Map == nil {
				return nil, fmt.Errorf("%w: section %d: url sections are not supported", ErrInvalidMap, i)
			}
			if s.Offset.Line < 0 || s.Offset.Column < 0 {
				return nil, fmt.Errorf("%w: section %d: negative offset", ErrInvalidMap, i)
			}
			sub, err := parse(*s.Map, depth+1)
			if err != nil {
				return nil, fmt.Errorf("section %d: %w", i, err)
			}
			if n := len(m.sections); n > 0 {
				p := m.sections[n-1]
				if s.Offset.Line < p.line || (s.Offset.Line == p.line && s.Offset.Column < p.col) {
					return nil, fmt.Errorf("%w: sections out of order", ErrInvalidMap)
				}
			}
			m.sections = append(m.sections, section{line: s.Offset.Line, col: s.Offset.Column, m: sub})
			m.size += sub.size
		}
		m.size += 64 + len(m.File)
		return m, nil
	}

	m.Sources = make([]string, len(r.Sources))
	for i, s := range r.Sources {
		m.Sources[i] = joinRoot(r.SourceRoot, s)
		m.size += len(s)
	}
	for _, c := range r.SourcesContent {
		if c != nil {
			m.size += len(*c)
		}
	}
	for _, n := range r.Names {
		m.size += len(n)
	}
	lines, err := decodeMappings(r.Mappings, len(m.Sources), len(m.Names))
	if err != nil {
		return nil, err
	}
	m.lines = lines
	for _, l := range lines {
		m.size += 24 + len(l)*20
	}
	m.size += 64 + len(m.File)
	return m, nil
}

func joinRoot(root, src string) string {
	if root == "" || strings.HasPrefix(src, "/") || strings.Contains(src, "://") {
		return src
	}
	return strings.TrimSuffix(root, "/") + "/" + src
}

func sortSegments(s []segment) {
	less := func(i, j int) bool { return s[i].genCol < s[j].genCol }
	if !sort.SliceIsSorted(s, less) {
		sort.SliceStable(s, less)
	}
}

func decodeMappings(s string, nSources, nNames int) ([][]segment, error) {
	var (
		lines                      [][]segment
		cur                        []segment
		src, srcLine, srcCol, name int
		genCol                     int
	)
	pos := 0
	for pos <= len(s) {
		if pos == len(s) {
			lines = append(lines, cur)
			break
		}
		switch s[pos] {
		case ';':
			sortSegments(cur)
			lines = append(lines, cur)
			cur, genCol = nil, 0
			pos++
			continue
		case ',':
			pos++
			continue
		}
		var fields [5]int
		n := 0
		for pos < len(s) && s[pos] != ',' && s[pos] != ';' {
			if n == 5 {
				return nil, fmt.Errorf("%w: segment with more than 5 fields", ErrInvalidMap)
			}
			v, np, err := decodeVLQ(s, pos)
			if err != nil {
				return nil, err
			}
			fields[n], pos, n = v, np, n+1
		}
		if n != 1 && n != 4 && n != 5 {
			return nil, fmt.Errorf("%w: segment with %d fields", ErrInvalidMap, n)
		}
		genCol += fields[0]
		if genCol < 0 {
			return nil, fmt.Errorf("%w: negative generated column", ErrInvalidMap)
		}
		seg := segment{genCol: int32(genCol), src: -1, name: -1}
		if n >= 4 {
			src += fields[1]
			srcLine += fields[2]
			srcCol += fields[3]
			if src < 0 || src >= nSources || srcLine < 0 || srcCol < 0 {
				return nil, fmt.Errorf("%w: source reference out of range", ErrInvalidMap)
			}
			seg.src, seg.srcLine, seg.srcCol = int32(src), int32(srcLine), int32(srcCol)
		}
		if n == 5 {
			name += fields[4]
			if name < 0 || name >= nNames {
				return nil, fmt.Errorf("%w: name reference out of range", ErrInvalidMap)
			}
			seg.name = int32(name)
		}
		cur = append(cur, seg)
	}
	if len(lines) > 0 {
		sortSegments(lines[len(lines)-1])
	}
	return lines, nil
}

// Size is an estimate of the memory held by the map, in bytes.
func (m *Map) Size() int { return m.size }

// Lookup resolves a generated position. line is 1-based, column is 0-based.
// It returns the mapping of the last segment starting at or before column on
// that line. ok is false when the line is out of range, no segment precedes
// the column, or the segment carries no source.
func (m *Map) Lookup(line, column int) (Mapping, bool) {
	if line < 1 || column < 0 {
		return Mapping{}, false
	}
	if m.sections != nil {
		l0 := line - 1
		i := sort.Search(len(m.sections), func(i int) bool {
			s := m.sections[i]
			return s.line > l0 || (s.line == l0 && s.col > column)
		}) - 1
		if i < 0 {
			return Mapping{}, false
		}
		s := m.sections[i]
		col := column
		if l0 == s.line {
			col -= s.col
		}
		return s.m.Lookup(l0-s.line+1, col)
	}
	if line > len(m.lines) {
		return Mapping{}, false
	}
	segs := m.lines[line-1]
	i := sort.Search(len(segs), func(i int) bool { return int(segs[i].genCol) > column }) - 1
	if i < 0 || segs[i].src < 0 {
		return Mapping{}, false
	}
	sg := segs[i]
	res := Mapping{Source: m.Sources[sg.src], Line: int(sg.srcLine) + 1, Column: int(sg.srcCol)}
	if sg.name >= 0 {
		res.Name = m.Names[sg.name]
	}
	return res, true
}

// SourceContent returns the embedded content of an original source, looked up
// by the (sourceRoot-resolved) path returned in Mapping.Source. For index maps
// the sections are searched in order.
func (m *Map) SourceContent(source string) (string, bool) {
	for _, s := range m.sections {
		if c, ok := s.m.SourceContent(source); ok {
			return c, true
		}
	}
	for i, s := range m.Sources {
		if s == source && i < len(m.SourcesContent) && m.SourcesContent[i] != nil {
			return *m.SourcesContent[i], true
		}
	}
	return "", false
}
