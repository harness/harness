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

var (
	ErrLineTooLong = errors.New("line too long")
)

func newUTF8Scanner(inner Scanner, modifier func([]byte) []byte) *utf8Scanner {
	return &utf8Scanner{
		scanner:  inner,
		modifier: modifier,
	}
}

// utf8Scanner is wrapping the provided scanner with UTF-8 checks and a modifier function.
type utf8Scanner struct {
	nextLine []byte
	nextErr  error

	modifier func([]byte) []byte
	scanner  Scanner
}

func (s *utf8Scanner) Scan() bool {
	scanOut := s.scanner.Scan()
	if !scanOut {
		s.nextLine = nil
		s.nextErr = s.scanner.Err()

		// to stay consistent with diff parser, treat bufio.ErrTooLong as binary file
		if errors.Is(s.nextErr, bufio.ErrTooLong) {
			s.nextErr = ErrBinaryFile
		}

		return false
	}

	// finalize next bytes
	original := s.scanner.Bytes()

	// Git is using first 8000 chars, but for now we stay consistent with diff parser
	// https://git.kernel.org/pub/scm/git/git.git/tree/xdiff-interface.c?h=v2.30.0#n187
	if !utf8.Valid(original) {
		s.nextLine = nil
		s.nextErr = ErrBinaryFile

		return false
	}

	// copy bytes to ensure nothing happens during modification
	cpy := make([]byte, len(original))
	copy(cpy, original)
	if s.modifier != nil {
		cpy = s.modifier(cpy)
	}

	s.nextLine = cpy
	s.nextErr = nil

	return true
}

func (s *utf8Scanner) Err() error {
	return s.nextErr
}

func (s *utf8Scanner) Bytes() []byte {
	return s.nextLine
}

func (s *utf8Scanner) Text() string {
	return string(s.nextLine)
}

// ReadTextFile returns a Scanner that reads the provided text file line by line.
//
// The returned Scanner fulfills the following:
//   - If any line is larger than 64kb, the scanning fails with ErrBinaryFile
//   - If the reader returns invalid UTF-8, the scanning fails with ErrBinaryFile
//   - Line endings are returned as-is, unless overwriteLE is provided
func ReadTextFile(r io.Reader, overwriteLE *string) (Scanner, string, error) {
	scanner := NewScannerWithPeek(r, ScanLinesWithEOF)
	peekOut := scanner.Peek()
	if !peekOut && scanner.Err() != nil {
		return nil, "", fmt.Errorf("unknown error while peeking first line: %w", scanner.Err())
	}

	// get raw bytes as we don't modify the slice
	firstLine := scanner.Bytes()

	// Heuristic - get line ending of file by first line, default to LF if there's no line endings in the file
	lineEnding := "\n"
	if HasLineEndingCRLF(firstLine) {
		lineEnding = "\r\n"
	}

	return newUTF8Scanner(scanner, func(line []byte) []byte {
		// overwrite line ending if requested (unless there's no line ending - e.g. last line)
		if overwriteLE != nil {
			if HasLineEndingCRLF(line) {
				return append(line[:len(line)-2], []byte(*overwriteLE)...)
			} else if HasLineEndingLF(line) {
				return append(line[:len(line)-1], []byte(*overwriteLE)...)
			}
		}

		return line
	}), lineEnding, nil
}

func HasLineEnding(line []byte) bool {
	// HasLineEndingLF is superset of HasLineEndingCRLF
	return HasLineEndingLF(line)
}

func HasLineEndingLF(line []byte) bool {
	return len(line) >= 1 && line[len(line)-1] == '\n'
}

func HasLineEndingCRLF(line []byte) bool {
	return len(line) >= 2 && line[len(line)-2] == '\r' && line[len(line)-1] == '\n'
}
