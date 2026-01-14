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

package pullreq

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

type suggestion struct {
	checkSum string
	code     string
}

// parseSuggestions parses the provided string for any markdown code blocks that are suggestions.
func parseSuggestions(s string) []suggestion {
	const languageSuggestion = "suggestion"

	out := []suggestion{}
	for len(s) > 0 {
		code, language, remaining, found := findNextMarkdownCodeBlock(s)

		// always update s to the remainder
		s = remaining

		if !found {
			break
		}

		if !strings.EqualFold(language, languageSuggestion) {
			continue
		}

		out = append(out,
			suggestion{
				checkSum: hashCodeBlock(code),
				code:     code,
			},
		)
	}

	return out
}

func hashCodeBlock(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

// findNextMarkdownCodeBlock finds a code block in markdown.
// NOTE: In the future we might want to use a proper markdown parser.
func findNextMarkdownCodeBlock(s string) (code string, language string, remaining string, found bool) {
	// find fenced code block header
	var startSequence string
	s = foreachLine(s, func(line string) bool {
		line, ok := trimMarkdownWhitespace(line)
		if !ok {
			return true
		}

		// try to find start sequence of a fenced code block (```+ or ~~~+)
		startSequence, line = cutLongestPrefix(line, '~')
		if len(startSequence) < 3 {
			startSequence, line = cutLongestPrefix(line, '`')
			if len(startSequence) < 3 {
				// no code block prefix found in this line
				return true
			}

			if strings.Contains(line, "`") {
				// any single tic in the same line breaks a code block ``` opening
				return true
			}
		}

		language = strings.TrimSpace(line)

		return false
	})

	if len(startSequence) == 0 {
		return "", "", "", false
	}

	// parse codeBuilder block
	codeBuilder := strings.Builder{}
	linesAdded := 0
	addLineToCode := func(line string) {
		// To normalize we:
		// - always use LF line ending
		// - strip any line ending from last line
		//
		// e.g. "```suggestion\n```" is the same as "```suggestion\n" is the same as "```suggestion"
		//
		// This ensures similar result with and without end markers for fenced code blocks,
		// and gives the user control on adding new lines at the end of the file.
		if linesAdded > 0 {
			codeBuilder.WriteByte('\n')
		}
		linesAdded++

		codeBuilder.WriteString(line)
	}

	s = foreachLine(s, func(line string) bool {
		// keep original line for appending it to code block if required
		originalLine := line

		line, ok := trimMarkdownWhitespace(line)
		if !ok {
			addLineToCode(originalLine)
			return true
		}

		if !strings.HasPrefix(line, startSequence) {
			addLineToCode(originalLine)
			return true
		}

		_, line = cutLongestPrefix(line, rune(startSequence[0])) // any higher number of chars as starting sequence works
		line = strings.TrimSpace(line)                           // spaces are fine
		if len(line) > 0 {
			// end of fenced code block can't contain anything else but spaces
			addLineToCode(originalLine)
			return true
		}

		return false
	})

	return codeBuilder.String(), language, s, true
}

// trimMarkdownWhitespace returns the provided line by removing any leading whitespaces.
// If the white space makes it an indented code block line, false is returned.
func trimMarkdownWhitespace(line string) (string, bool) {
	// remove any leading spaces
	prefix, updatedLine := cutLongestPrefix(line, ' ')
	if len(prefix) >= 4 {
		// line is considered a code line by indentation
		return line, false
	}

	// check for leading tabs (doesn't matter how many)
	if strings.HasPrefix(updatedLine, "\t") {
		// line is considered a code line by indentation
		return line, false
	}

	return updatedLine, true
}

// foreachLine iterates over the provided string and calls "process" method for each line.
// If process returns false, or the scan reaches the end of the lines, the scanning stops.
// The method returns the remaining text of s.
func foreachLine(s string, process func(line string) bool) string {
	for len(s) > 0 {
		line, remaining, _ := strings.Cut(s, "\n")

		// always update s to the remaining string
		s = remaining

		// handle CLRF
		if lineLen := len(line); lineLen > 0 && line[lineLen-1] == '\r' {
			line = line[:lineLen-1]
		}

		if !process(line) {
			return s
		}
	}

	return s
}

// cutLongestPrefix returns the longest prefix of repeating 'c' together with the remainder of the string.
func cutLongestPrefix(s string, c rune) (string, string) {
	if len(s) == 0 {
		return "", ""
	}

	i := strings.IndexFunc(s, func(r rune) bool { return r != c })
	if i < 0 {
		// no character found that's different from the provided rune!
		return s, ""
	}

	return s[:i], s[i:]
}
