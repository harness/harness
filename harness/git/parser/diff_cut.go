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

package parser

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

type DiffFileHeader struct {
	OldFileName string
	NewFileName string
	Extensions  map[string]string
}

type DiffCutParams struct {
	LineStart    int
	LineStartNew bool
	LineEnd      int
	LineEndNew   bool
	BeforeLines  int
	AfterLines   int
	LineLimit    int
}

// DiffCut parses git diff output that should consist of a single hunk
// (usually generated with large value passed to the "--unified" parameter)
// and returns lines specified with the parameters.
//
//nolint:funlen,gocognit,nestif,gocognit,gocyclo,cyclop // it's actually very readable
func DiffCut(r io.Reader, params DiffCutParams) (HunkHeader, Hunk, error) {
	scanner := bufio.NewScanner(r)

	var err error
	var hunkHeader HunkHeader

	if _, err = scanFileHeader(scanner); err != nil {
		return HunkHeader{}, Hunk{}, err
	}

	if hunkHeader, err = scanHunkHeader(scanner); err != nil {
		return HunkHeader{}, Hunk{}, err
	}

	currentOldLine := hunkHeader.OldLine
	currentNewLine := hunkHeader.NewLine

	var inCut bool
	var diffCutHeader HunkHeader
	var diffCut []string

	linesBeforeBuf := newStrCircBuf(params.BeforeLines)

	for {
		if params.LineEndNew && currentNewLine > params.LineEnd ||
			!params.LineEndNew && currentOldLine > params.LineEnd {
			break // exceeded the requested line range
		}

		var line string
		var action diffAction

		line, action, err = scanHunkLine(scanner)
		if err != nil {
			return HunkHeader{}, Hunk{}, err
		}

		if line == "" {
			err = io.EOF
			break
		}

		if params.LineStartNew && currentNewLine < params.LineStart ||
			!params.LineStartNew && currentOldLine < params.LineStart {
			// not yet in the requested line range
			linesBeforeBuf.push(line)
		} else {
			if !inCut {
				diffCutHeader.NewLine = currentNewLine
				diffCutHeader.OldLine = currentOldLine
			}
			inCut = true

			if action != actionRemoved {
				diffCutHeader.NewSpan++
			}
			if action != actionAdded {
				diffCutHeader.OldSpan++
			}

			diffCut = append(diffCut, line)
			if params.LineLimit > 0 && len(diffCut) >= params.LineLimit {
				break // safety break
			}
		}

		// increment the line numbers
		if action != actionRemoved {
			currentNewLine++
		}
		if action != actionAdded {
			currentOldLine++
		}
	}

	if !inCut {
		return HunkHeader{}, Hunk{}, ErrHunkNotFound
	}

	var (
		linesBefore []string
		linesAfter  []string
	)

	linesBefore = linesBeforeBuf.lines()
	if !errors.Is(err, io.EOF) {
		for i := 0; i < params.AfterLines; i++ {
			line, _, err := scanHunkLine(scanner)
			if err != nil {
				return HunkHeader{}, Hunk{}, err
			}
			if line == "" {
				break
			}
			linesAfter = append(linesAfter, line)
		}
	}

	diffCutHeaderLines := diffCutHeader

	for _, s := range linesBefore {
		action := diffAction(s[0])
		if action != actionRemoved {
			diffCutHeaderLines.NewLine--
			diffCutHeaderLines.NewSpan++
		}
		if action != actionAdded {
			diffCutHeaderLines.OldLine--
			diffCutHeaderLines.OldSpan++
		}
	}

	for _, s := range linesAfter {
		action := diffAction(s[0])
		if action != actionRemoved {
			diffCutHeaderLines.NewSpan++
		}
		if action != actionAdded {
			diffCutHeaderLines.OldSpan++
		}
	}

	return diffCutHeader, Hunk{
		HunkHeader: diffCutHeaderLines,
		Lines:      concat(linesBefore, diffCut, linesAfter),
	}, nil
}

// scanFileHeader keeps reading lines until file header line is read.
func scanFileHeader(scan *bufio.Scanner) (DiffFileHeader, error) {
	for scan.Scan() {
		line := scan.Text()
		if h, ok := ParseDiffFileHeader(line); ok {
			return h, nil
		}
	}

	if err := scan.Err(); err != nil {
		return DiffFileHeader{}, err
	}

	return DiffFileHeader{}, ErrHunkNotFound
}

// scanHunkHeader keeps reading lines until hunk header line is read.
func scanHunkHeader(scan *bufio.Scanner) (HunkHeader, error) {
	for scan.Scan() {
		line := scan.Text()
		if h, ok := ParseDiffHunkHeader(line); ok {
			return h, nil
		}
	}

	if err := scan.Err(); err != nil {
		return HunkHeader{}, err
	}

	return HunkHeader{}, ErrHunkNotFound
}

type diffAction byte

const (
	actionUnchanged diffAction = ' '
	actionRemoved   diffAction = '-'
	actionAdded     diffAction = '+'
)

func scanHunkLine(scan *bufio.Scanner) (line string, action diffAction, err error) {
again:
	action = actionUnchanged

	if !scan.Scan() {
		err = scan.Err()
		return
	}

	line = scan.Text()
	if line == "" {
		err = ErrHunkNotFound // should not happen: empty line in diff output
		return
	}

	action = diffAction(line[0])
	if action == '\\' { // handle the "\ No newline at end of file" line
		goto again
	}

	if action != actionRemoved && action != actionAdded && action != actionUnchanged {
		// treat this as the end of hunk
		line = ""
		action = actionUnchanged
		return
	}

	return
}

// BlobCut parses raw file and returns lines specified with the parameter.
func BlobCut(r io.Reader, params DiffCutParams) (CutHeader, Cut, error) {
	scanner := bufio.NewScanner(r)

	var (
		err               error
		lineNumber        int
		inCut             bool
		cutStart, cutSpan int
		cutLines          []string
	)

	extStart := params.LineStart - params.BeforeLines
	extEnd := params.LineEnd + params.AfterLines
	linesNeeded := params.LineEnd - params.LineStart + 1

	for {
		if !scanner.Scan() {
			err = scanner.Err()
			break
		}

		lineNumber++
		line := scanner.Text()

		if !utf8.ValidString(line) {
			return CutHeader{}, Cut{}, ErrBinaryFile
		}

		if lineNumber > extEnd {
			break // exceeded the requested line range
		}

		if lineNumber < extStart {
			// not yet in the requested line range
			continue
		}

		if !inCut {
			cutStart = lineNumber
			inCut = true
		}
		cutLines = append(cutLines, line)
		cutSpan++

		if lineNumber >= params.LineStart && lineNumber <= params.LineEnd {
			linesNeeded--
		}

		if params.LineLimit > 0 && len(cutLines) >= params.LineLimit {
			break
		}
	}

	if errors.Is(err, bufio.ErrTooLong) {
		// By default, the max token size is 65536 (bufio.MaxScanTokenSize).
		// If the file contains a line that is longer than this we treat it as a binary file.
		return CutHeader{}, Cut{}, ErrBinaryFile
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return CutHeader{}, Cut{}, fmt.Errorf("failed to parse blob cut: %w", err)
	}

	if !inCut || linesNeeded > 0 {
		return CutHeader{}, Cut{}, ErrHunkNotFound
	}

	// the cut header is hunk-like header (with Line and Span) that describes the requested lines exactly
	ch := CutHeader{Line: params.LineStart, Span: params.LineEnd - params.LineStart + 1}

	// the cut includes the requested lines and few more lines specified with the BeforeLines and AfterLines.
	c := Cut{CutHeader: CutHeader{Line: cutStart, Span: cutSpan}, Lines: cutLines}

	return ch, c, nil
}

func LimitLineLen(lines *[]string, maxLen int) {
outer:
	for idxLine, line := range *lines {
		var l int
		for idxRune := range line {
			l++
			if l > maxLen {
				(*lines)[idxLine] = line[:idxRune] + "…" // append the ellipsis to indicate that the line was trimmed.
				continue outer
			}
		}
	}
}

type strCircBuf struct {
	head    int
	entries []string
}

func newStrCircBuf(size int) strCircBuf {
	return strCircBuf{
		head:    -1,
		entries: make([]string, 0, size),
	}
}

func (b *strCircBuf) push(s string) {
	n := cap(b.entries)
	if n == 0 {
		return
	}

	b.head++

	if len(b.entries) < n {
		b.entries = append(b.entries, s)
		return
	}

	if b.head >= n {
		b.head = 0
	}
	b.entries[b.head] = s
}

func (b *strCircBuf) lines() []string {
	n := cap(b.entries)
	if len(b.entries) < n {
		return b.entries
	}

	res := make([]string, n)
	for i := 0; i < n; i++ {
		idx := (b.head + 1 + i) % n
		res[i] = b.entries[idx]
	}
	return res
}

func concat[T any](a ...[]T) []T {
	var n int
	for _, m := range a {
		n += len(m)
	}
	res := make([]T, n)

	n = 0
	for _, m := range a {
		copy(res[n:], m)
		n += len(m)
	}

	return res
}
