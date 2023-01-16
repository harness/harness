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
	ErrEmptyLeftCommitID           = errors.New("empty LeftCommitId")
	ErrEmptyRightCommitID          = errors.New("empty RightCommitId")
)

// MergeConflictsError represents an error if merging fails with a conflict.
type MergeConflictsError struct {
	Method string
	StdOut string
	StdErr string
	Err    error
}

func (err MergeConflictsError) Error() string {
	return fmt.Sprintf("Merge Conflict Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

// MergeUnrelatedHistoriesError represents an error if merging fails due to unrelated histories.
type MergeUnrelatedHistoriesError struct {
	Method string
	StdOut string
	StdErr string
	Err    error
}

func (err MergeUnrelatedHistoriesError) Error() string {
	return fmt.Sprintf("Merge UnrelatedHistories Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}
