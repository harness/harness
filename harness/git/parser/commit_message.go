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
	"strings"
	"unicode"
)

// CleanUpWhitespace removes extra whitespace for the multiline string passed as parameter.
// The intended usage is to clean up commit messages.
func CleanUpWhitespace(message string) string {
	var messageStarted bool
	var isLastLineEmpty bool

	const eol = '\n'

	builder := strings.Builder{}

	scan := bufio.NewScanner(strings.NewReader(message))
	for scan.Scan() {
		line := strings.TrimRightFunc(scan.Text(), unicode.IsSpace)

		if len(line) == 0 {
			if messageStarted {
				isLastLineEmpty = true
			}
			continue
		}

		if isLastLineEmpty {
			builder.WriteByte(eol)
		}

		builder.WriteString(line)
		builder.WriteByte(eol)
		isLastLineEmpty = false
		messageStarted = true
	}

	return builder.String()
}

// ExtractSubject extracts subject from a commit message. The result should be like output of
// the one line commit summary, like "git log --oneline" or "git log --format=%s".
func ExtractSubject(message string) string {
	var messageStarted bool

	builder := strings.Builder{}

	scan := bufio.NewScanner(strings.NewReader(message))
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())

		// process empty lines
		if len(line) == 0 {
			if messageStarted {
				return builder.String()
			}
			continue
		}

		if messageStarted {
			builder.WriteByte(' ')
		}

		builder.WriteString(line)
		messageStarted = true
	}

	return builder.String()
}

// SplitMessage splits a commit message. Returns two strings:
// * subject (the one line commit summary, like "git log --oneline" or "git log --format=%s),
// * body only (like "git log --format=%b").
func SplitMessage(message string) (string, string) {
	var state int
	var lastLineEmpty bool
	const (
		stateInit = iota
		stateSubject
		stateSeparator
		stateBody
	)

	const eol = '\n'

	subjectBuilder := strings.Builder{}
	bodyBuilder := strings.Builder{}

	scan := bufio.NewScanner(strings.NewReader(message))
	for scan.Scan() {
		line := strings.TrimRightFunc(scan.Text(), unicode.IsSpace)

		// process empty lines
		if len(line) == 0 {
			switch state {
			case stateInit, stateSeparator:
				// ignore all empty lines before the first line of the subject
			case stateSubject:
				state = stateSeparator
			case stateBody:
				lastLineEmpty = true
			}
			continue
		}

		switch state {
		case stateInit:
			state = stateSubject
			subjectBuilder.WriteString(strings.TrimLeftFunc(line, unicode.IsSpace))
		case stateSubject:
			subjectBuilder.WriteByte(' ')
			subjectBuilder.WriteString(strings.TrimLeftFunc(line, unicode.IsSpace))
		case stateSeparator:
			state = stateBody
			bodyBuilder.WriteString(line)
			bodyBuilder.WriteByte(eol)
			lastLineEmpty = false
		case stateBody:
			if lastLineEmpty {
				bodyBuilder.WriteByte(eol)
			}
			bodyBuilder.WriteString(line)
			bodyBuilder.WriteByte(eol)
			lastLineEmpty = false
		}
	}

	return subjectBuilder.String(), bodyBuilder.String()
}
