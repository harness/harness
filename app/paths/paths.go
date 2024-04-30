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
)

var (
	ErrPathEmpty = errors.New("path is empty")
)

// DisectLeaf splits a path into its parent path and the leaf name
// e.g. space1/space2/space3 -> (space1/space2, space3, nil).
func DisectLeaf(path string) (string, string, error) {
	path = strings.Trim(path, types.PathSeparator)

	if path == "" {
		return "", "", ErrPathEmpty
	}

	i := strings.LastIndex(path, types.PathSeparator)
	if i == -1 {
		return "", path, nil
	}

	return path[:i], path[i+1:], nil
}

// DisectRoot splits a path into its root space and sub-path
// e.g. space1/space2/space3 -> (space1, space2/space3, nil).
func DisectRoot(path string) (string, string, error) {
	path = strings.Trim(path, types.PathSeparator)

	if path == "" {
		return "", "", ErrPathEmpty
	}

	i := strings.Index(path, types.PathSeparator)
	if i == -1 {
		return path, "", nil
	}

	return path[:i], path[i+1:], nil
}

/*
 * Concatenate two paths together (takes care of leading / trailing '/')
 * e.g. (space1/, /space2/) -> space1/space2
 *
 * NOTE: "//" is not a valid path, so all '/' will be trimmed.
 */
func Concatenate(path1 string, path2 string) string {
	path1 = strings.Trim(path1, types.PathSeparator)
	path2 = strings.Trim(path2, types.PathSeparator)

	if path1 == "" {
		return path2
	} else if path2 == "" {
		return path1
	}

	return path1 + types.PathSeparator + path2
}

// Segments returns all segments of the path
// e.g. /space1/space2/space3 -> [space1, space2, space3].
func Segments(path string) []string {
	path = strings.Trim(path, types.PathSeparator)
	return strings.Split(path, types.PathSeparator)
}

// IsAncesterOf returns true iff 'path' is an ancestor of 'other' or they are the same.
// e.g. other = path(/.*).
func IsAncesterOf(path string, other string) bool {
	path = strings.Trim(path, types.PathSeparator)
	other = strings.Trim(other, types.PathSeparator)

	// add "/" to both to handle space1/inner and space1/in
	return strings.Contains(
		other+types.PathSeparator,
		path+types.PathSeparator,
	)
}

// Parent returns the parent path of the provided path.
// if the path doesn't have a parent an empty string is returned.
func Parent(path string) string {
	spacePath, _, _ := DisectLeaf(path)
	return spacePath
}
