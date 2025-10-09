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
	"bytes"
	"io"
)

type Scanner interface {
	Scan() bool
	Err() error
	Bytes() []byte
	Text() string
}

func ScanZeroSeparated(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil // Return nothing if at end of file and no data passed
	}
	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[0:i], nil // Split at zero byte
	}
	if atEOF {
		return len(data), data, nil // at the end of file return the data
	}
	return
}

// ScanLinesWithEOF is a variation of bufio's ScanLine method that returns the line endings.
// https://cs.opensource.google/go/go/+/master:src/bufio/scan.go;l=355;drc=bc2124dab14fa292e18df2937037d782f7868635
func ScanLinesWithEOF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data[:i+1], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func NewScannerWithPeek(r io.Reader, split bufio.SplitFunc) *ScannerWithPeek {
	scanner := bufio.NewScanner(r)
	scanner.Split(split)
	return &ScannerWithPeek{
		scanner: scanner,
	}
}

type ScannerWithPeek struct {
	peeked        bool
	peekedScanOut bool

	nextLine []byte
	nextErr  error

	scanner *bufio.Scanner
}

func (s *ScannerWithPeek) scan() bool {
	scanOut := s.scanner.Scan()

	s.nextErr = s.scanner.Err()
	s.nextLine = s.scanner.Bytes()

	return scanOut
}

func (s *ScannerWithPeek) Peek() bool {
	if s.peeked {
		s.nextLine = nil
		s.nextErr = ErrPeekedMoreThanOnce

		return false
	}

	// load next line
	scanOut := s.scan()

	// set peeked data
	s.peeked = true
	s.peekedScanOut = scanOut

	return scanOut
}

func (s *ScannerWithPeek) Scan() bool {
	if s.peeked {
		s.peeked = false
		return s.peekedScanOut
	}

	return s.scan()
}

func (s *ScannerWithPeek) Err() error {
	return s.nextErr
}

func (s *ScannerWithPeek) Bytes() []byte {
	return s.nextLine
}

func (s *ScannerWithPeek) Text() string {
	return string(s.nextLine)
}
