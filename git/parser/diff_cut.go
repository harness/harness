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
	"io"

	"github.com/harness/gitness/git/types"
)

// DiffCut parses git diff output that should consist of a single hunk
// (usually generated with large value passed to the "--unified" parameter)
// and returns lines specified with the parameters.
//
//nolint:funlen,gocognit,nestif,gocognit,gocyclo,cyclop // it's actually very readable
func DiffCut(r io.Reader, params types.DiffCutParams) (types.HunkHeader, types.Hunk, error) {
	scanner := bufio.NewScanner(r)

	var err error
	var hunkHeader types.HunkHeader

	if _, err = scanFileHeader(scanner); err != nil {
		return types.HunkHeader{}, types.Hunk{}, err
	}

	if hunkHeader, err = scanHunkHeader(scanner); err != nil {
		return types.HunkHeader{}, types.Hunk{}, err
	}

	currentOldLine := hunkHeader.OldLine
	currentNewLine := hunkHeader.NewLine

	var inCut bool
	var diffCutHeader types.HunkHeader
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
			return types.HunkHeader{}, types.Hunk{}, err
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
			if len(diffCut) > params.LineLimit {
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
		return types.HunkHeader{}, types.Hunk{}, types.ErrHunkNotFound
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
				return types.HunkHeader{}, types.Hunk{}, err
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

	return diffCutHeader, types.Hunk{
		HunkHeader: diffCutHeaderLines,
		Lines:      concat(linesBefore, diffCut, linesAfter),
	}, nil
}

// scanFileHeader keeps reading lines until file header line is read.
func scanFileHeader(scan *bufio.Scanner) (types.DiffFileHeader, error) {
	for scan.Scan() {
		line := scan.Text()
		if h, ok := ParseDiffFileHeader(line); ok {
			return h, nil
		}
	}

	if err := scan.Err(); err != nil {
		return types.DiffFileHeader{}, err
	}

	return types.DiffFileHeader{}, types.ErrHunkNotFound
}

// scanHunkHeader keeps reading lines until hunk header line is read.
func scanHunkHeader(scan *bufio.Scanner) (types.HunkHeader, error) {
	for scan.Scan() {
		line := scan.Text()
		if h, ok := ParseDiffHunkHeader(line); ok {
			return h, nil
		}
	}

	if err := scan.Err(); err != nil {
		return types.HunkHeader{}, err
	}

	return types.HunkHeader{}, types.ErrHunkNotFound
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
		err = types.ErrHunkNotFound // should not happen: empty line in diff output
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
