// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	minDisplayNameLength = 1
	maxDisplayNameLength = 256

	minUIDLength = 2
	maxUIDLength = 64
	uidRegex     = "^[a-zA-Z_][a-zA-Z0-9-_]*$"

	minEmailLength = 1
	maxEmailLength = 250
)

var (
	ErrDisplayNameLength = &ValidationError{
		fmt.Sprintf("DisplayName has to be between %d and %d in length.", minDisplayNameLength, maxDisplayNameLength),
	}
	ErrDisplayNameContainsInvalidASCII = &ValidationError{"DisplayName has to consist of valid ASCII characters."}

	ErrUIDLength = &ValidationError{
		fmt.Sprintf("UID has to be between %d and %d in length.",
			minUIDLength, maxUIDLength),
	}
	ErrUIDRegex = &ValidationError{
		"UID has to start with a letter (or _) and only contain the following characters [a-zA-Z0-9-_].",
	}

	ErrEmailLen = &ValidationError{
		fmt.Sprintf("Email address has to be within %d and %d characters", minEmailLength, maxEmailLength),
	}
)

// DisplayName checks the provided display name and returns an error if it isn't valid.
func DisplayName(displayName string) error {
	l := len(displayName)
	if l < minDisplayNameLength || l > maxDisplayNameLength {
		return ErrDisplayNameLength
	}

	// created sanitized string restricted to ASCII characters (without control characters).
	sanitizedString := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r > 255 {
			return -1
		}
		return r
	}, displayName)

	if len(sanitizedString) != len(displayName) {
		return ErrDisplayNameContainsInvalidASCII
	}

	return nil
}

// UID checks the provided uid and returns an error if it isn't valid.
func UID(uid string) error {
	l := len(uid)
	if l < minUIDLength || l > maxUIDLength {
		return ErrUIDLength
	}

	if ok, _ := regexp.Match(uidRegex, []byte(uid)); !ok {
		return ErrUIDRegex
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
