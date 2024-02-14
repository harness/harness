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
	"fmt"
	"io"
	"regexp"
)

type DiffStatus byte

const (
	DiffStatusModified = 'M'
	DiffStatusAdded    = 'A'
	DiffStatusDeleted  = 'D'
	DiffStatusRenamed  = 'R'
	DiffStatusCopied   = 'C'
	DiffStatusType     = 'T'
)

type DiffRawFile struct {
	OldBlobSHA string
	NewBlobSHA string
	Status     byte
	OldPath    string
	Path       string
}

var regexpDiffRaw = regexp.MustCompile(`:\d{6} \d{6} ([0-9a-f]+) ([0-9a-f]+) (\w)(\d*)`)

// DiffRaw parses raw git diff output (git diff --raw). Each entry (a line) is a changed file.
// The format is:
//
//	:100644 100644 <old-hash> <new-hash> <status>NULL<file-name>NULL
//
// Old-hash and new-hash are the file object hashes. Status can be A added, D deleted, M modified, R renamed, C copied.
// When the status is A then the old-hash is the zero hash, when the status is D the new-hash is the zero hash.
// If the status is R or C then the output is:
//
//	:100644 100644 <old-hash> <new-hash> R<similarity index>NULL<old-name>NULL<new-name>NULL
func DiffRaw(r io.Reader) ([]DiffRawFile, error) {
	var result []DiffRawFile

	scan := bufio.NewScanner(r)
	scan.Split(ScanZeroSeparated)
	for scan.Scan() {
		s := scan.Text()
		groups := regexpDiffRaw.FindStringSubmatch(s)
		if groups == nil {
			continue
		}

		var oldPath, path string

		if !scan.Scan() {
			return nil, fmt.Errorf("failed to get path for the entry: %q; err=%w", s, scan.Err())
		}

		path = scan.Text()

		status := groups[3][0]
		switch status {
		case DiffStatusRenamed, DiffStatusCopied:
			if !scan.Scan() {
				return nil, fmt.Errorf("failed to get new path for the entry: %q; err=%w", s, scan.Err())
			}

			oldPath, path = path, scan.Text()
		case DiffStatusAdded, DiffStatusDeleted, DiffStatusModified, DiffStatusType:
		default:
			return nil, fmt.Errorf("got invalid raw diff status=%c for entry %s %s", status, s, path)
		}

		result = append(result, DiffRawFile{
			OldBlobSHA: groups[1],
			NewBlobSHA: groups[2],
			Status:     status,
			OldPath:    oldPath,
			Path:       path,
		})
	}
	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan raw diff: %w", scan.Err())
	}

	return result, nil
}
