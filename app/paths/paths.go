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

package paths

import (
	"errors"
	"strings"

	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
)

var (
	ErrPathEmpty = errors.New("path is empty")
)

// DisectLeaf splits a path into its parent path and the leaf name
// e.g. space1/space2/space3 -> (space1/space2, space3, nil).
func DisectLeaf(path string) (string, string, error) {
	path = strings.Trim(path, types.PathSeparatorAsString)

	if path == "" {
		return "", "", ErrPathEmpty
	}

	i := strings.LastIndex(path, types.PathSeparatorAsString)
	if i == -1 {
		return "", path, nil
	}

	return path[:i], path[i+1:], nil
}

// DisectRoot splits a path into its root space and sub-path
// e.g. space1/space2/space3 -> (space1, space2/space3, nil).
func DisectRoot(path string) (string, string, error) {
	path = strings.Trim(path, types.PathSeparatorAsString)

	if path == "" {
		return "", "", ErrPathEmpty
	}

	i := strings.Index(path, types.PathSeparatorAsString)
	if i == -1 {
		return path, "", nil
	}

	return path[:i], path[i+1:], nil
}

/*
 * Concatenate two paths together (takes care of leading / trailing '/')
 * e.g. (space1/, /space2/) -> space1/space2
 *
 * NOTE: All leading, trailing, and consecutive '/' will be trimmed to ensure correct paths.
 */
func Concatenate(paths ...string) string {
	if len(paths) == 0 {
		return ""
	}

	sb := strings.Builder{}
	for i := 0; i < len(paths); i++ {
		// remove all leading, trailing, and consecutive '/'
		var nextRune *rune
		for _, r := range paths[i] {
			// skip '/' if we already have '/' as next rune or we don't have anything other than '/' yet
			if (nextRune == nil || *nextRune == types.PathSeparator) && r == types.PathSeparator {
				continue
			}

			// first time we take a rune we have to make sure we add a separator if needed (guaranteed no '/').
			if nextRune == nil && sb.Len() > 0 {
				sb.WriteString(types.PathSeparatorAsString)
			}

			// flush the previous rune before storing the next rune
			if nextRune != nil {
				sb.WriteRune(*nextRune)
			}

			nextRune = ptr.Of(r)
		}

		// flush the final rune
		if nextRune != nil && *nextRune != types.PathSeparator {
			sb.WriteRune(*nextRune)
		}
	}

	return sb.String()
}

// Segments returns all segments of the path
// e.g. space1/space2/space3 -> [space1, space2, space3].
func Segments(path string) []string {
	path = strings.Trim(path, types.PathSeparatorAsString)
	return strings.Split(path, types.PathSeparatorAsString)
}

// Depth returns the depth of the path.
// e.g. space1/space2 -> 2.
func Depth(path string) int {
	path = strings.Trim(path, types.PathSeparatorAsString)
	if len(path) == 0 {
		return 0
	}

	return strings.Count(path, types.PathSeparatorAsString) + 1
}

// IsAncesterOf returns true iff 'path' is an ancestor of 'other' or they are the same.
// e.g. other = path(/.*).
func IsAncesterOf(path string, other string) bool {
	path = strings.Trim(path, types.PathSeparatorAsString)
	other = strings.Trim(other, types.PathSeparatorAsString)

	// add "/" to both to handle space1/inner and space1/in
	return strings.Contains(
		other+types.PathSeparatorAsString,
		path+types.PathSeparatorAsString,
	)
}

// Parent returns the parent path of the provided path.
// if the path doesn't have a parent an empty string is returned.
func Parent(path string) string {
	spacePath, _, _ := DisectLeaf(path)
	return spacePath
}
