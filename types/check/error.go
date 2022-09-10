// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

var (
	ErrAny = &CheckError{}
)

/*
 * An error returned by check methods for any validation errors
 * WARNING: This error will be printed to the user as is!
 */
type CheckError struct {
	msg string
}

func (e *CheckError) Error() string {
	return e.msg
}

func (e *CheckError) Is(target error) bool {
	// If the caller is checking for any CheckError, return true
	if target == ErrAny {
		return true
	}

	// ensure it's the correct type
	v, ok := target.(*CheckError)
	if !ok {
		return false
	}

	// only the same if the message is the same
	return e.msg == v.msg
}
