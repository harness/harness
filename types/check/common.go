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

	"github.com/harness/gitness/types"
)

const (
	minDisplayNameLength = 1
	maxDisplayNameLength = 256

	minIdentifierLength              = 1
	MaxIdentifierLength              = 100
	identifierRegex                  = "^[a-zA-Z0-9-_.]*$"
	illegalRepoSpaceIdentifierSuffix = ".git"

	minEmailLength = 1
	maxEmailLength = 250

	maxDescriptionLength = 1024
)

var (
	// illegalRootSpaceIdentifiers is the list of space identifier we are blocking for root spaces
	// as they might cause issues with routing.
	illegalRootSpaceIdentifiers = []string{"api", "git"}
)

var (
	ErrDisplayNameLength = &ValidationError{
		fmt.Sprintf("DisplayName has to be between %d and %d in length.", minDisplayNameLength, maxDisplayNameLength),
	}

	ErrDescriptionTooLong = &ValidationError{
		fmt.Sprintf("Description can be at most %d in length.", maxDescriptionLength),
	}

	ErrIdentifierLength = &ValidationError{
		fmt.Sprintf(
			"Identifier has to be between %d and %d in length.",
			minIdentifierLength,
			MaxIdentifierLength,
		),
	}
	ErrIdentifierRegex = &ValidationError{
		"Identifier can only contain the following characters [a-zA-Z0-9-_.].",
	}

	ErrEmailLen = &ValidationError{
		fmt.Sprintf("Email address has to be within %d and %d characters", minEmailLength, maxEmailLength),
	}

	ErrInvalidCharacters = &ValidationError{"Input contains invalid characters."}

	ErrIllegalRootSpaceIdentifier = &ValidationError{
		fmt.Sprintf("The following identifiers are not allowed for a root space: %v", illegalRootSpaceIdentifiers),
	}

	ErrIllegalRepoSpaceIdentifierSuffix = &ValidationError{
		fmt.Sprintf("Space and repository identifiers cannot end with %q.", illegalRepoSpaceIdentifierSuffix),
	}

	ErrIllegalPrincipalUID = &ValidationError{
		fmt.Sprintf("Principal UID is not allowed to be %q.", types.AnonymousPrincipalUID),
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

// Identifier checks the provided identifier and returns an error if it isn't valid.
func Identifier(identifier string) error {
	l := len(identifier)
	if l < minIdentifierLength || l > MaxIdentifierLength {
		return ErrIdentifierLength
	}

	if ok, _ := regexp.Match(identifierRegex, []byte(identifier)); !ok {
		return ErrIdentifierRegex
	}

	return nil
}

type RepoIdentifier func(identifier string) error

// RepoIdentifierDefault performs the default Identifier check and also blocks illegal repo identifiers.
func RepoIdentifierDefault(identifier string) error {
	if err := Identifier(identifier); err != nil {
		return err
	}

	identifierLower := strings.ToLower(identifier)
	if strings.HasSuffix(identifierLower, illegalRepoSpaceIdentifierSuffix) {
		return ErrIllegalRepoSpaceIdentifierSuffix
	}

	return nil
}

// PrincipalUID is an abstraction of a validation method that verifies principal UIDs.
// NOTE: Enables support for different principal UID formats.
type PrincipalUID func(uid string) error

// PrincipalUIDDefault performs the default Principal UID check.
func PrincipalUIDDefault(uid string) error {
	if err := Identifier(uid); err != nil {
		return err
	}

	if strings.EqualFold(uid, types.AnonymousPrincipalUID) {
		return ErrIllegalPrincipalUID
	}

	return nil
}

// SpaceIdentifier is an abstraction of a validation method that returns true
// iff the Identifier is valid to be used in a resource path for repo/space.
// NOTE: Enables support for different path formats.
type SpaceIdentifier func(identifier string, isRoot bool) error

// SpaceIdentifierDefault performs the default Identifier check and also blocks illegal root space Identifiers.
func SpaceIdentifierDefault(identifier string, isRoot bool) error {
	if err := Identifier(identifier); err != nil {
		return err
	}

	identifierLower := strings.ToLower(identifier)
	if strings.HasSuffix(identifierLower, illegalRepoSpaceIdentifierSuffix) {
		return ErrIllegalRepoSpaceIdentifierSuffix
	}

	if isRoot {
		for _, p := range illegalRootSpaceIdentifiers {
			if p == identifierLower {
				return ErrIllegalRootSpaceIdentifier
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
