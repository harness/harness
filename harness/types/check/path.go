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

package check

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/types"
)

const (
	MaxSpacePathDepth = 9
	MaxRepoPathDepth  = 10
)

var (
	ErrPathEmpty = &ValidationError{
		"Path can't be empty.",
	}
	ErrPathInvalidDepth = &ValidationError{
		fmt.Sprintf("A path can have at most %d segments (%d for spaces).",
			MaxRepoPathDepth, MaxSpacePathDepth),
	}
	ErrEmptyPathSegment = &ValidationError{
		"Empty segments are not allowed.",
	}
	ErrPathCantBeginOrEndWithSeparator = &ValidationError{
		fmt.Sprintf("Path can't start or end with the separator ('%s').", types.PathSeparatorAsString),
	}
)

// Path checks the provided path and returns an error in it isn't valid.
func Path(path string, isSpace bool, identifierCheck SpaceIdentifier) error {
	if path == "" {
		return ErrPathEmpty
	}

	// ensure path doesn't begin or end with /
	if path[:1] == types.PathSeparatorAsString || path[len(path)-1:] == types.PathSeparatorAsString {
		return ErrPathCantBeginOrEndWithSeparator
	}

	// ensure path is not too deep
	if err := PathDepth(path, isSpace); err != nil {
		return err
	}

	// ensure all segments of the path are valid identifiers
	segments := strings.Split(path, types.PathSeparatorAsString)
	for i, s := range segments {
		if s == "" {
			return ErrEmptyPathSegment
		} else if err := identifierCheck(s, i == 0); err != nil {
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
	l := strings.Count(path, types.PathSeparatorAsString) + 1
	return (!isSpace && l > MaxRepoPathDepth) || (isSpace && l > MaxSpacePathDepth)
}
