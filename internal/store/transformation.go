// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package store

import (
	"strings"
)

// PrincipalUIDTransformation transforms a principalUID to a value that should be duplicate free.
// This allows us to simply switch between principalUIDs being case sensitive, insensitive or anything in between.
type PrincipalUIDTransformation func(uid string) (string, error)

func ToLowerPrincipalUIDTransformation(uid string) (string, error) {
	return strings.ToLower(uid), nil
}

// SpacePathTransformation transforms a path to a value that should be duplicate free.
// This allows us to simply switch between paths being case sensitive, insensitive or anything in between.
type SpacePathTransformation func(original string, isRoot bool) string

func ToLowerSpacePathTransformation(original string, _ bool) string {
	return strings.ToLower(original)
}
