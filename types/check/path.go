// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/types"
	"github.com/pkg/errors"
)

const (
	minPathSegments         = 1
	maxPathSegmentsForSpace = 9
	maxPathSegments         = 10
)

var (
	ErrPathEmpty                       = &CheckError{"Path can't be empty."}
	ErrPathInvalidSize                 = &CheckError{fmt.Sprintf("A path has to be between %d and %d segments long (%d for spaces).", minPathSegments, maxPathSegments, maxPathSegmentsForSpace)}
	ErrEmptyPathSegment                = &CheckError{"Empty segments are not allowed."}
	ErrPathCantBeginOrEndWithSeparator = &CheckError{fmt.Sprintf("Path can't start or end with the separator ('%s').", types.PathSeparator)}
	ErrPathDifferentTopLevelSpace      = &CheckError{"Alias paths have to stay within the same top level space."}
	ErrTopLevelPathNotAllowed          = &CheckError{"Top level alias paths are not allowed."}
)

/*
 * Path checks the provided path and returns an error in it isn't valid.
 *
 * NOTE: A repository path can be one deeper than a space path (as otherwise the space would be useless)
 */
func Path(path string, isSpace bool) error {

	if path == "" {
		return ErrPathEmpty
	}

	// ensure path doesn't begin or end with /
	if path[:1] == types.PathSeparator || path[len(path)-1:] == types.PathSeparator {
		return ErrPathCantBeginOrEndWithSeparator
	}

	// ensure path is not too deep
	segments := strings.Split(path, types.PathSeparator)
	l := len(segments)
	if l < minPathSegments || (isSpace == false && l > maxPathSegments) || (isSpace && l > maxPathSegmentsForSpace) {
		return ErrPathInvalidSize
	}

	// ensure all segments of the path are valid
	for _, s := range segments {
		if s == "" {
			return ErrEmptyPathSegment
		} else if err := Name(s); err != nil {
			return errors.Wrapf(err, "Invalid segment '%s'", s)
		}
	}

	return nil
}

/*
 * Validates a PathParams object that is used to create a new path.
 *
 * NOTES:
 *	- We don't allow top level alias paths
 *	- An alias path has to stay within the same top level space
 *
 * IMPORTANT:
 *	Technically there can be a racing condition when a space is being moved inbetween the validation and path creation.
 *	But that is fine, as the path could've also been created a second earlier when it was still valid and would then still exist.
 */
func PathParams(path *types.PathParams, currentPath string, isSpace bool) error {
	// ensure the path is valid
	if err := Path(path.Path, isSpace); err != nil {
		return err
	}

	// ensure the path is at least 1 level deep (at least one '/')
	i := strings.Index(path.Path, types.PathSeparator)
	if i < 0 {
		return ErrTopLevelPathNotAllowed
	}

	// ensure the top level space doesn't change (add path separator to avoid abcd -> abc matching)
	if !strings.HasPrefix(currentPath+types.PathSeparator, path.Path[:i]+types.PathSeparator) {
		return ErrPathDifferentTopLevelSpace
	}

	return nil
}

/*
 * Checks if the provided path is too long.
 *
 * NOTE: A repository path can be one deeper than a space path (as otherwise the space would be useless)
 */
func PathTooLong(path string, isSpace bool) bool {
	l := strings.Count(path, types.PathSeparator) + 1
	return (isSpace == false && l > maxPathSegments) || (isSpace && l > maxPathSegmentsForSpace)
}
