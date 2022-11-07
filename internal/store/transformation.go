// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package store

import (
	"strings"

	"github.com/harness/gitness/types/enum"
)

// PrincipalUIDTransformation transforms a principalUID to a value that should be duplicate free.
// This allows us to simply switch between principalUIDs being case sensitive, insensitive or anything inbetween.
type PrincipalUIDTransformation func(principalType enum.PrincipalType, uid string) (string, error)

func ToLowerPrincipalUIDTransformation(principalType enum.PrincipalType, uid string) (string, error) {
	return strings.ToLower(uid), nil
}

// PathTransformation transforms a path to a value that should be duplicate free.
// This allows us to simply switch between paths being case sensitive, insensitive or anything inbetween.
type PathTransformation func(string) (string, error)

func ToLowerPathTransformation(original string) (string, error) {
	return strings.ToLower(original), nil
}
