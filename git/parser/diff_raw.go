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
	"strconv"

	"github.com/harness/gitness/git/sha"
)

type DiffStatus byte

const (
	DiffStatusModified DiffStatus = 'M'
	DiffStatusAdded    DiffStatus = 'A'
	DiffStatusDeleted  DiffStatus = 'D'
	DiffStatusRenamed  DiffStatus = 'R'
	DiffStatusCopied   DiffStatus = 'C'
	DiffStatusType     DiffStatus = 'T'
)

func (s DiffStatus) String() string {
	return fmt.Sprintf("%c", s)
}

type DiffRawFile struct {
	OldFileMode string
	NewFileMode string
	OldBlobSHA  string
	NewBlobSHA  string
	Status      DiffStatus
	OldPath     string
	Path        string
}

var regexpDiffRaw = regexp.MustCompile(`:(\d{6}) (\d{6}) ([0-9a-f]+) ([0-9a-f]+) (\w)(\d*)`)

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

		status := DiffStatus(groups[5][0])
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
			OldFileMode: groups[1],
			NewFileMode: groups[2],
			OldBlobSHA:  groups[3],
			NewBlobSHA:  groups[4],
			Status:      status,
			OldPath:     oldPath,
			Path:        path,
		})
	}
	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan raw diff: %w", scan.Err())
	}

	return result, nil
}

type BatchCheckObject struct {
	SHA sha.SHA
	// TODO: Use proper TreeNodeType
	Type string
	Size int64
}

var regexpBatchCheckObject = regexp.MustCompile(`^([0-9a-f]{40,64}) (\w+) (\d+)$`)

func CatFileBatchCheckAllObjects(r io.Reader) ([]BatchCheckObject, error) {
	var result []BatchCheckObject

	scan := bufio.NewScanner(r)
	scan.Split(ScanZeroSeparated)

	for scan.Scan() {
		line := scan.Text()
		matches := regexpBatchCheckObject.FindStringSubmatch(line)

		if len(matches) != 4 {
			return nil, fmt.Errorf("failed to parse line: %q", line)
		}

		sha, err := sha.New(matches[1])
		if err != nil {
			return nil, fmt.Errorf("failed to create sha.SHA for %q: %w", matches[1], err)
		}

		sizeStr := matches[3]
		size, err := strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert size %q to int64: %w", sizeStr, err)
		}

		result = append(result, BatchCheckObject{
			SHA:  sha,
			Type: matches[2],
			Size: size,
		})
	}
	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan cat file batch check all objects: %w", scan.Err())
	}

	return result, nil
}
