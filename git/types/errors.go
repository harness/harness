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

package types

import (
	"fmt"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/enum"
)

const (
	StatusNotMergeable errors.Status = "not_mergeable"
)

var (
	ErrAlreadyExists   = errors.Conflict("already exists")
	ErrInvalidPath     = errors.NotFound("path is invalid")
	ErrSHADoesNotMatch = errors.InvalidArgument("sha does not match")
	ErrNoDefaultBranch = errors.NotFound("no default branch")
	ErrHunkNotFound    = errors.NotFound("hunk not found")
)

type NotFoundError struct {
	Msg string
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, &NotFoundError{})
}

func (e *NotFoundError) Error() string {
	return e.Msg
}

//nolint:errorlint // the purpose of this method is to check whether the target itself if of this type.
func (e *NotFoundError) Is(target error) bool {
	_, ok := target.(*NotFoundError)
	return ok
}

func ErrNotFound(format string, args ...any) error {
	return &NotFoundError{
		Msg: fmt.Sprintf(format, args...),
	}
}

type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
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
