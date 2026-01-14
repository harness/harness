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
	"regexp"
	"strconv"
	"strings"
)

type Hunk struct {
	HunkHeader
	Lines []string
}

type HunkHeader struct {
	OldLine int
	OldSpan int
	NewLine int
	NewSpan int
	Text    string
}

type Cut struct {
	CutHeader
	Lines []string
}

type CutHeader struct {
	Line int
	Span int
}

var regExpHunkHeader = regexp.MustCompile(`^@@ -([0-9]+)(,([0-9]+))? \+([0-9]+)(,([0-9]+))? @@( (.+))?$`)

func (h *HunkHeader) IsZero() bool {
	return h.OldLine == 0 && h.OldSpan == 0 && h.NewLine == 0 && h.NewSpan == 0
}

func (h *HunkHeader) IsValid() bool {
	oldOk := h.OldLine == 0 && h.OldSpan == 0 || h.OldLine > 0 && h.OldSpan > 0
	newOk := h.NewLine == 0 && h.NewSpan == 0 || h.NewLine > 0 && h.NewSpan > 0
	return !h.IsZero() && oldOk && newOk
}

func (h *HunkHeader) String() string {
	sb := strings.Builder{}

	sb.WriteString("@@ -")

	sb.WriteString(strconv.Itoa(h.OldLine))
	if h.OldSpan != 1 {
		sb.WriteByte(',')
		sb.WriteString(strconv.Itoa(h.OldSpan))
	}

	sb.WriteString(" +")

	sb.WriteString(strconv.Itoa(h.NewLine))
	if h.NewSpan != 1 {
		sb.WriteByte(',')
		sb.WriteString(strconv.Itoa(h.NewSpan))
	}

	sb.WriteString(" @@")

	if h.Text != "" {
		sb.WriteByte(' ')
		sb.WriteString(h.Text)
	}

	return sb.String()
}

func ParseDiffHunkHeader(line string) (HunkHeader, bool) {
	groups := regExpHunkHeader.FindStringSubmatch(line)
	if groups == nil {
		return HunkHeader{}, false
	}

	oldLine, _ := strconv.Atoi(groups[1])
	oldSpan := 1
	if groups[3] != "" {
		oldSpan, _ = strconv.Atoi(groups[3])
	}

	newLine, _ := strconv.Atoi(groups[4])
	newSpan := 1
	if groups[6] != "" {
		newSpan, _ = strconv.Atoi(groups[6])
	}

	return HunkHeader{
		OldLine: oldLine,
		OldSpan: oldSpan,
		NewLine: newLine,
		NewSpan: newSpan,
		Text:    groups[8],
	}, true
}
