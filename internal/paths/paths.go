// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package paths

import (
	"errors"
	"strings"

	"github.com/harness/gitness/types"
)

var (
	ErrPathEmpty = errors.New("path is empty")
)

// Disect splits a path into its parent path and the leaf name
// e.g. /space1/space2/space3 -> (/space1/space2, space3, nil).
func Disect(path string) (string, string, error) {
	if path == "" {
		return "", "", ErrPathEmpty
	}

	i := strings.LastIndex(path, types.PathSeparator)
	if i == -1 {
		return "", path, nil
	}

	return path[:i], path[i+1:], nil
}

/*
 * Concatinate two paths together (takes care of leading / trailing '/')
 * e.g. (space1/, /space2/) -> space1/space2
 *
 * NOTE: "//" is not a valid path, so all '/' will be trimmed.
 */
func Concatinate(path1 string, path2 string) string {
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
	return strings.Split(path, types.PathSeparator)
}
