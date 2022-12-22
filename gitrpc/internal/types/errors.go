// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

import "errors"

var (
	ErrAlreadyExists           = errors.New("already exists")
	ErrInvalidArgument         = errors.New("invalid argument")
	ErrNotFound                = errors.New("not found")
	ErrInvalidPath             = errors.New("path is invalid")
	ErrUndefinedAction         = errors.New("undefined action")
	ErrContentSentBeforeAction = errors.New("content sent before action")
	ErrActionListEmpty         = errors.New("no commit actions to perform on repository")
	ErrHeaderCannotBeEmpty     = errors.New("header field cannot be empty")
	ErrSHADoesNotMatch         = errors.New("sha does not match")
	ErrEmptyLeftCommitID       = errors.New("empty LeftCommitId")
	ErrEmptyRightCommitID      = errors.New("empty RightCommitId")
)
