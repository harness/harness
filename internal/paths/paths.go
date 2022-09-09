// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package paths

import (
	"strings"

	"github.com/harness/gitness/internal/errs"
	"github.com/harness/gitness/types"
)

/*
 * Splits a path into its parent path and the leaf name.
 * e.g. /space1/space2/space3 -> (/space1/space2, space3, nil)
 */
func Disect(path string) (string, string, error) {
	if path == "" {
		return "", "", errs.PathEmpty
	}

	i := strings.LastIndex(path, types.PathSeparator)
	if i == -1 {
		return "", path, nil
	}

	return path[:i], path[i+1:], nil
}

/*
 * Concatinates two paths together (takes care of leading / trailing '/')
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
