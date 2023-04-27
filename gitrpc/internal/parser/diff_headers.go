// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package parser

import (
	"bufio"
	"io"
	"regexp"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/internal/types"
)

var regExpDiffFileHeader = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)

func ParseDiffFileHeader(line string) (types.DiffFileHeader, bool) {
	groups := regExpDiffFileHeader.FindStringSubmatch(line)
	if groups == nil {
		return types.DiffFileHeader{}, false
	}

	return types.DiffFileHeader{
		OldFileName: groups[1],
		NewFileName: groups[2],
		Extensions:  map[string]string{},
	}, true
}

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

func ParseDiffFileExtendedHeader(line string) (string, string) {
	groups := regExpDiffExtHeader.FindStringSubmatch(line)
	if groups == nil {
		return "", ""
	}

	return groups[1], groups[2]
}

// GetHunkHeaders parses git diff output and returns all diff headers for all files.
// See for documentation: https://git-scm.com/docs/git-diff#generate_patch_text_with_p
func GetHunkHeaders(r io.Reader) ([]*types.DiffFileHunkHeaders, error) {
	scanner := bufio.NewScanner(r)

	var currentFile *types.DiffFileHunkHeaders
	var result []*types.DiffFileHunkHeaders

	for scanner.Scan() {
		line := scanner.Text()

		if h, ok := ParseDiffFileHeader(line); ok {
			if currentFile != nil {
				result = append(result, currentFile)
			}
			currentFile = &types.DiffFileHunkHeaders{
				FileHeader:   h,
				HunksHeaders: nil,
			}

			continue
		}

		if currentFile == nil {
			// should not happen: we reached the hunk header without first finding the file header.
			return nil, types.ErrHunkNotFound
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
