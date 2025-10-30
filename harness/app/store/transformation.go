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
