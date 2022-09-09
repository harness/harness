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
	ErrPathEmpty                       = fmt.Errorf("Path can't be empty.")
	ErrPathInvalidSize                 = fmt.Errorf("A path has to be between %d and %d segments long (%d for spaces).", minPathSegments, maxPathSegments, maxPathSegmentsForSpace)
	ErrEmptyPathSegment                = fmt.Errorf("Empty segments are not allowed.")
	ErrPathCantBeginOrEndWithSeparator = fmt.Errorf("Path can't start or end with the separator ('%s').", types.PathSeparator)
)

/*
 * PathParams checks the provided path params and returns an error in it isn't valid.
 *
 * NOTE: A repository path can be one deeper than a space path (as otherwise the space would be useless)
 */
func PathParams(path string, isSpace bool) error {

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
