// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"errors"
	"fmt"
)

var (
	ErrAny = &ValidationError{}
)

// ValidationError is error returned for any validation errors.
// WARNING: This error will be printed to the user as is!
type ValidationError struct {
	msg string
}

func NewValidationError(msg string) *ValidationError {
	return &ValidationError{
		msg: msg,
	}
}

func NewValidationErrorf(format string, args ...interface{}) *ValidationError {
	return &ValidationError{
		msg: fmt.Sprintf(format, args...),
	}
}

func (e *ValidationError) Error() string {
	return e.msg
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
	return e.msg == err.msg
}
