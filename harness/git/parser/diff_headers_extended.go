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

	"github.com/harness/gitness/git/enum"
)

var regExpDiffExtHeader = regexp.MustCompile(
	"^(" +
		enum.DiffExtHeaderOldMode + "|" +
		enum.DiffExtHeaderNewMode + "|" +
		enum.DiffExtHeaderDeletedFileMode + "|" +
		enum.DiffExtHeaderNewFileMode + "|" +
		enum.DiffExtHeaderCopyFrom + "|" +
		enum.DiffExtHeaderCopyTo + "|" +
		enum.DiffExtHeaderRenameFrom + "|" +
		enum.DiffExtHeaderRenameTo + "|" +
		enum.DiffExtHeaderSimilarity + "|" +
		enum.DiffExtHeaderDissimilarity + "|" +
		enum.DiffExtHeaderIndex +
		") (.+)$")

// ParseDiffFileExtendedHeader parses a generic extended header line.
func ParseDiffFileExtendedHeader(line string) (string, string) {
	groups := regExpDiffExtHeader.FindStringSubmatch(line)
	if groups == nil {
		return "", ""
	}

	return groups[1], groups[2]
}

// regExpDiffFileIndexHeader parses the `index` extended header line with a format like:
//
//	index f994c2cf569523ba736473bbfbac3700fa1db28d..0000000000000000000000000000000000000000
//	index 68233d6cd204b0df84e91a1ce8c8b75e13529973..e69de29bb2d1d6434b8b29ae775ad8c2e48c5391 100644
//
// NOTE: it's NEW_SHA..OLD_SHA.
var regExpDiffFileIndexHeader = regexp.MustCompile(`^index ([0-9a-f]{4,64})\.\.([0-9a-f]{4,64})( [0-9]+)?$`)

// DiffExtParseIndex parses the `index` extended diff header line.
func DiffExtHeaderParseIndex(line string) (newSHA string, oldSHA string, ok bool) {
	groups := regExpDiffFileIndexHeader.FindStringSubmatch(line)
	if groups == nil {
		return "", "", false
	}

	return groups[1], groups[2], true
}
