// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import "errors"

var (
	ErrAny = &ValidationError{}
)

// ValidationError is error returned by check methods for any validation errors
// WARNING: This error will be printed to the user as is!
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

func (e *ValidationError) Is(target error) bool {
	// If the caller is checking for any ValidationError, return true
	if errors.Is(target, ErrAny) {
		return true
	}

	// ensure it's the correct type
	err := &ValidationError{}
	if !errors.As(target, &err) {
		return false
	}

	// only the same if the message is the same
	return e.Msg == err.Msg
}
