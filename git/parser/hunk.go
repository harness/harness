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

	"github.com/harness/gitness/git/types"
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
