// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package parser

import (
	"github.com/harness/gitness/gitrpc/internal/types"
	"regexp"
	"strconv"
)

var regExpHunkHeader = regexp.MustCompile(`^@@ -([0-9]+)(,([0-9]+))? \+([0-9]+)(,([0-9]+))? @@( (.+))?$`)

func ParseDiffHunkHeader(line string) (types.HunkHeader, bool) {
	groups := regExpHunkHeader.FindStringSubmatch(line)
	if groups == nil {
		return types.HunkHeader{}, false
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

	return types.HunkHeader{
		OldLine: oldLine,
		OldSpan: oldSpan,
		NewLine: newLine,
		NewSpan: newSpan,
		Text:    groups[8],
	}, true
}
