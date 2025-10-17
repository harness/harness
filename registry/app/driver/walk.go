// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// ErrSkipDir is used as a return value from onFileFunc to indicate that
// the directory named in the call is to be skipped. It is not returned
// as an error by any function.
var ErrSkipDir = errors.New("skip this directory")

// ErrFilledBuffer is used as a return value from onFileFunc to indicate
// that the requested number of entries has been reached and the walk can
// stop.
var ErrFilledBuffer = errors.New("we have enough entries")

// WalkFn is called once per file by Walk.
type WalkFn func(fileInfo FileInfo) error

// WalkFallback traverses a filesystem defined within driver, starting
// from the given path, calling f on each file. It uses the List method and Stat to drive itself.
// If the returned error from the WalkFn is ErrSkipDir the directory will not be entered and Walk
// will continue the traversal. If the returned error from the WalkFn is ErrFilledBuffer, the walk
// stops.
func WalkFallback(
	ctx context.Context,
	driver StorageDriver,
	from string,
	f WalkFn,
	options ...func(*WalkOptions),
) error {
	walkOptions := &WalkOptions{}
	for _, o := range options {
		o(walkOptions)
	}

	startAfterHint := walkOptions.StartAfterHint
	// Ensure that we are checking the hint is contained within from by adding a "/".
	// Add to both in case the hint and form are the same, which would still count.
	rel, err := filepath.Rel(from, startAfterHint)
	if err != nil || strings.HasPrefix(rel, "..") {
		// The startAfterHint is outside from, so check if we even need to walk anything
		// Replace any path separators with \x00 so that the sort works in a depth-first way
		if strings.ReplaceAll(startAfterHint, "/", "\x00") < strings.ReplaceAll(from, "/", "\x00") {
			_, err := doWalkFallback(ctx, driver, from, "", f)
			return err
		}
		return nil
	}
	// The startAfterHint is within from.
	// Walk up the tree until we hit from - we know it is contained.
	// Ensure startAfterHint is never deeper than a child of the base
	// directory so that doWalkFallback doesn't have to worry about
	// depth-first comparisons
	base := startAfterHint
	for strings.HasPrefix(base, from) {
		_, err = doWalkFallback(ctx, driver, base, startAfterHint, f)
		if (!errors.As(err, &PathNotFoundError{}) && err != nil) {
			return err
		}
		if base == from {
			break
		}
		startAfterHint = base
		base, _ = filepath.Split(startAfterHint)
		if len(base) > 1 {
			base = strings.TrimSuffix(base, "/")
		}
	}
	return nil
}

// doWalkFallback performs a depth first walk using recursion.
// from is the directory that this iteration of the function should walk.
// startAfterHint is the child within from to start the walk after.
// It should only ever be a child of from, or the empty string.
func doWalkFallback(
	ctx context.Context,
	driver StorageDriver,
	from string,
	startAfterHint string,
	f WalkFn,
) (bool, error) {
	children, err := driver.List(ctx, from)
	if err != nil {
		return false, err
	}
	sort.Strings(children)
	for _, child := range children {
		// The startAfterHint has been sanitised in WalkFallback and will either be
		// empty, or be suitable for an <= check for this _from_.
		if child <= startAfterHint {
			continue
		}

		fileInfo, err := driver.Stat(ctx, child)
		if err != nil {
			if errors.As(err, &PathNotFoundError{}) {
				// repository was removed in between listing and enumeration. Ignore it.
				log.Ctx(ctx).Info().Interface("path", child).Msg("ignoring deleted path")
			} else {
				return false, err
			}
		}
		err = f(fileInfo)
		switch {
		case err == nil && fileInfo.IsDir():
			if ok, err := doWalkFallback(ctx, driver, child, startAfterHint, f); err != nil || !ok {
				return ok, err
			}
		case errors.Is(err, ErrFilledBuffer):
			return false, nil // no error but stop iteration
		case err != nil:
			return false, err
		}
	}
	return true, nil
}
