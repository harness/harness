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
	"io"
	"regexp"
)

type DiffFileHunkHeaders struct {
	FileHeader   DiffFileHeader
	HunksHeaders []HunkHeader
}

var regExpDiffFileHeader = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)

func ParseDiffFileHeader(line string) (DiffFileHeader, bool) {
	groups := regExpDiffFileHeader.FindStringSubmatch(line)
	if groups == nil {
		return DiffFileHeader{}, false
	}

	return DiffFileHeader{
		OldFileName: groups[1],
		NewFileName: groups[2],
		Extensions:  map[string]string{},
	}, true
}

// GetHunkHeaders parses git diff output and returns all diff headers for all files.
// See for documentation: https://git-scm.com/docs/git-diff#generate_patch_text_with_p
func GetHunkHeaders(r io.Reader) ([]*DiffFileHunkHeaders, error) {
	scanner := bufio.NewScanner(r)

	var currentFile *DiffFileHunkHeaders
	var result []*DiffFileHunkHeaders

	for scanner.Scan() {
		line := scanner.Text()

		if h, ok := ParseDiffFileHeader(line); ok {
			if currentFile != nil {
				result = append(result, currentFile)
			}
			currentFile = &DiffFileHunkHeaders{
				FileHeader:   h,
				HunksHeaders: nil,
			}

			continue
		}

		if currentFile == nil {
			// should not happen: we reached the hunk header without first finding the file header.
			return nil, ErrHunkNotFound
		}

		if h, ok := ParseDiffHunkHeader(line); ok {
			currentFile.HunksHeaders = append(currentFile.HunksHeaders, h)
			continue
		}

		if headerKey, headerValue := ParseDiffFileExtendedHeader(line); headerKey != "" {
			currentFile.FileHeader.Extensions[headerKey] = headerValue
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if currentFile != nil {
		result = append(result, currentFile)
	}

	return result, nil
}
