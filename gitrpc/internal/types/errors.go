// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"errors"
	"fmt"

	"github.com/harness/gitness/gitrpc/enum"
)

var (
	ErrAlreadyExists               = errors.New("already exists")
	ErrInvalidArgument             = errors.New("invalid argument")
	ErrRepositoryNotFound          = errors.New("repository not found")
	ErrRepositoryCorrupted         = errors.New("repository corrupted")
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
	ErrNoDefaultBranch             = errors.New("no default branch")
	ErrFailedToConnect             = errors.New("failed to connect")
	ErrHunkNotFound                = errors.New("hunk not found")
	ErrEmptySHA                    = errors.New("empty SHA")
)

// MergeConflictsError represents an error if merging fails with a conflict.
type MergeConflictsError struct {
	Method    enum.MergeMethod
	CommitSHA string
	StdOut    string
	StdErr    string
	Err       error
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
	Method enum.MergeMethod
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

// PathNotFoundError represents an error if a path in a repo can't be found.
type PathNotFoundError struct {
	Path string
}

func IsPathNotFoundError(err error) bool {
	return errors.Is(err, &PathNotFoundError{})
}

func (e *PathNotFoundError) Error() string {
	return fmt.Sprintf("path '%s' wasn't found in the repo", e.Path)
}

func (e *PathNotFoundError) Unwrap() error {
	return nil
}

//nolint:errorlint // the purpose of this method is to check whether the target itself if of this type.
func (e *PathNotFoundError) Is(target error) bool {
	_, ok := target.(*PathNotFoundError)
	return ok
}
