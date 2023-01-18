// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/types"
)

const (
	maxPathSegmentsForSpace = 9
	maxPathSegments         = 10
)

var (
	ErrPathEmpty = &ValidationError{
		"Path can't be empty.",
	}
	ErrPathInvalidDepth = &ValidationError{
		fmt.Sprintf("A path can have at most %d segments (%d for spaces).",
			maxPathSegments, maxPathSegmentsForSpace),
	}
	ErrEmptyPathSegment = &ValidationError{
		"Empty segments are not allowed.",
	}
	ErrPathCantBeginOrEndWithSeparator = &ValidationError{
		fmt.Sprintf("Path can't start or end with the separator ('%s').", types.PathSeparator),
	}
)

// Path checks the provided path and returns an error in it isn't valid.
func Path(path string, isSpace bool, uidCheck PathUID) error {
	if path == "" {
		return ErrPathEmpty
	}

	// ensure path doesn't begin or end with /
	if path[:1] == types.PathSeparator || path[len(path)-1:] == types.PathSeparator {
		return ErrPathCantBeginOrEndWithSeparator
	}

	// ensure path is not too deep
	if err := PathDepth(path, isSpace); err != nil {
		return err
	}

	// ensure all segments of the path are valid uids
	segments := strings.Split(path, types.PathSeparator)
	for i, s := range segments {
		if s == "" {
			return ErrEmptyPathSegment
		} else if err := uidCheck(s, i == 0); err != nil {
			return fmt.Errorf("invalid segment '%s': %w", s, err)
		}
	}

	return nil
}

// PathDepth Checks the depth of the provided path.
func PathDepth(path string, isSpace bool) error {
	if IsPathTooDeep(path, isSpace) {
		return ErrPathInvalidDepth
	}

	return nil
}

// IsPathTooDeep Checks if the provided path is too long.
// NOTE: A repository path can be one deeper than a space path (as otherwise the space would be useless).
func IsPathTooDeep(path string, isSpace bool) bool {
	l := strings.Count(path, types.PathSeparator) + 1
	return (!isSpace && l > maxPathSegments) || (isSpace && l > maxPathSegmentsForSpace)
}
