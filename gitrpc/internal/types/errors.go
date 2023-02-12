// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import (
	"errors"
	"fmt"
)

var (
	ErrAlreadyExists               = errors.New("already exists")
	ErrInvalidArgument             = errors.New("invalid argument")
	ErrNotFound                    = errors.New("not found")
	ErrInvalidPath                 = errors.New("path is invalid")
	ErrUndefinedAction             = errors.New("undefined action")
	ErrActionNotAllowedOnEmptyRepo = errors.New("action not allowed on empty repository")
	ErrContentSentBeforeAction     = errors.New("content sent before action")
	ErrActionListEmpty             = errors.New("no commit actions to perform on repository")
	ErrHeaderCannotBeEmpty         = errors.New("header field cannot be empty")
	ErrBaseCannotBeEmpty           = errors.New("base field cannot be empty")
	ErrSHADoesNotMatch             = errors.New("sha does not match")
	ErrEmptyBaseRef                = errors.New("empty base reference")
	ErrEmptyHeadRef                = errors.New("empty head reference")
)

// MergeConflictsError represents an error if merging fails with a conflict.
type MergeConflictsError struct {
	Method string
	StdOut string
	StdErr string
	Err    error
}

func IsMergeConflictsError(err error) bool {
	return errors.Is(err, &MergeConflictsError{})
}

func (e *MergeConflictsError) Error() string {
	return fmt.Sprintf("Merge Conflict Error: %v: %s\n%s", e.Err, e.StdErr, e.StdOut)
}

func (e *MergeConflictsError) Unwrap() error {
	return e.Err
}

//nolint:errorlint // the purpose of this method is to check whether the target itself if of this type.
func (e *MergeConflictsError) Is(target error) bool {
	_, ok := target.(*MergeConflictsError)
	return ok
}

// MergeUnrelatedHistoriesError represents an error if merging fails due to unrelated histories.
type MergeUnrelatedHistoriesError struct {
	Method string
	StdOut string
	StdErr string
	Err    error
}

func IsMergeUnrelatedHistoriesError(err error) bool {
	return errors.Is(err, &MergeUnrelatedHistoriesError{})
}

func (e *MergeUnrelatedHistoriesError) Error() string {
	return fmt.Sprintf("Merge UnrelatedHistories Error: %v: %s\n%s", e.Err, e.StdErr, e.StdOut)
}

func (e *MergeUnrelatedHistoriesError) Unwrap() error {
	return e.Err
}

//nolint:errorlint // the purpose of this method is to check whether the target itself if of this type.
func (e *MergeUnrelatedHistoriesError) Is(target error) bool {
	_, ok := target.(*MergeUnrelatedHistoriesError)
	return ok
}
