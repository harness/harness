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
	"regexp"
	"strings"
)

const (
	minDisplayNameLength = 1
	maxDisplayNameLength = 256

	minUIDLength = 1
	MaxUIDLength = 100
	uidRegex     = "^[a-zA-Z_][a-zA-Z0-9-_.]*$"

	minEmailLength = 1
	maxEmailLength = 250

	maxDescriptionLength = 1024
)

var (
	// illegalRootSpaceUIDs is the list of space UIDs we are blocking for root spaces
	// as they might cause issues with routing.
	illegalRootSpaceUIDs = []string{"api", "git"}
)

var (
	ErrDisplayNameLength = &ValidationError{
		fmt.Sprintf("DisplayName has to be between %d and %d in length.", minDisplayNameLength, maxDisplayNameLength),
	}

	ErrDescriptionTooLong = &ValidationError{
		fmt.Sprintf("Description can be at most %d in length.", maxDescriptionLength),
	}

	ErrUIDLength = &ValidationError{
		fmt.Sprintf("UID has to be between %d and %d in length.",
			minUIDLength, MaxUIDLength),
	}
	ErrUIDRegex = &ValidationError{
		"UID has to start with a letter (or _) and only contain the following characters [a-zA-Z0-9-_.].",
	}

	ErrEmailLen = &ValidationError{
		fmt.Sprintf("Email address has to be within %d and %d characters", minEmailLength, maxEmailLength),
	}

	ErrInvalidCharacters = &ValidationError{"Input contains invalid characters."}

	ErrIllegalRootSpaceUID = &ValidationError{
		fmt.Sprintf("The following names are not allowed for a root space: %v", illegalRootSpaceUIDs),
	}
)

// DisplayName checks the provided display name and returns an error if it isn't valid.
func DisplayName(displayName string) error {
	l := len(displayName)
	if l < minDisplayNameLength || l > maxDisplayNameLength {
		return ErrDisplayNameLength
	}

	return ForControlCharacters(displayName)
}

// Description checks the provided description and returns an error if it isn't valid.
func Description(description string) error {
	l := len(description)
	if l > maxDescriptionLength {
		return ErrDescriptionTooLong
	}

	return ForControlCharacters(description)
}

// ForControlCharacters ensures that there are no control characters in the provided string.
func ForControlCharacters(s string) error {
	for _, r := range s {
		if r < 32 || r == 127 {
			return ErrInvalidCharacters
		}
	}

	return nil
}

// UID checks the provided uid and returns an error if it isn't valid.
func UID(uid string) error {
	l := len(uid)
	if l < minUIDLength || l > MaxUIDLength {
		return ErrUIDLength
	}

	if ok, _ := regexp.Match(uidRegex, []byte(uid)); !ok {
		return ErrUIDRegex
	}

	return nil
}

// PrincipalUID is an abstraction of a validation method that verifies principal UIDs.
// NOTE: Enables support for different principal UID formats.
type PrincipalUID func(uid string) error

// PrincipalUIDDefault performs the default Principal UID check.
func PrincipalUIDDefault(uid string) error {
	return UID(uid)
}

// PathUID is an abstraction of a validation method that returns true
// iff the UID is valid to be used in a resource path for repo/space.
// NOTE: Enables support for different path formats.
type PathUID func(uid string, isRoot bool) error

// PathUIDDefault performs the default UID check and also blocks illegal root space UIDs.
func PathUIDDefault(uid string, isRoot bool) error {
	if err := UID(uid); err != nil {
		return err
	}

	if isRoot {
		uidLower := strings.ToLower(uid)
		for _, p := range illegalRootSpaceUIDs {
			if p == uidLower {
				return ErrIllegalRootSpaceUID
			}
		}
	}

	return nil
}

// Email checks the provided email and returns an error if it isn't valid.
func Email(email string) error {
	l := len(email)
	if l < minEmailLength || l > maxEmailLength {
		return ErrEmailLen
	}

	// TODO: add better email validation.

	return nil
}
