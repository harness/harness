// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package errs

import (
	"fmt"
)

// Static errors
var (
	// Indicates that a requested resource wasn't found.
	ResourceNotFound error = &dynamicError{0, "Resource not found", nil}
	Duplicate        error = &dynamicError{1, "Resource is a duplicate", nil}
	PathTooLong      error = &dynamicError{2, "The path is too long", nil}
)

// Wrappers
func WrapInResourceNotFound(inner error) error {
	return cloneWithNewInner(ResourceNotFound.(*dynamicError), inner)
}
func WrapInDuplicate(inner error) error {
	return cloneWithNewInner(Duplicate.(*dynamicError), inner)
}
func WrapInPathTooLongf(format string, args ...interface{}) error {
	return cloneWithNewMsg(PathTooLong.(*dynamicError), fmt.Sprintf(format, args...))
}

// Error type (on purpose not using explicit definitions and iota, to make overhead as small as possible)
type dynamicErrorType int

/*
 * This is an abstraction of an error that can be both a standalone error or a wrapping error.
 * The idea is to allow errors.Is(err, errs.MyError) for wrapping errors while keeping code to a minimum
 */
type dynamicError struct {
	errorType dynamicErrorType
	msg       string
	inner     error
}

func (e *dynamicError) Error() string {
	if e.inner == nil {
		return e.msg
	} else {
		return fmt.Sprintf("%s: %s", e.msg, e.inner)
	}
}
func (e *dynamicError) Unwrap() error {
	return e.inner
}

func (e *dynamicError) Is(target error) bool {
	te, ok := target.(*dynamicError)
	return ok && te.errorType == e.errorType
}

func cloneWithNewMsg(d *dynamicError, msg string) *dynamicError {
	return &dynamicError{d.errorType, msg, nil}
}

func cloneWithNewInner(d *dynamicError, inner error) *dynamicError {
	return &dynamicError{d.errorType, d.msg, inner}
}

func cloneWithNewMsgAndInner(d *dynamicError, msg string, inner error) *dynamicError {
	return &dynamicError{d.errorType, msg, inner}
}
