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

package api

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/enum"

	"github.com/rs/zerolog/log"
)

var (
	ErrAlreadyExists       = errors.New("already exists")
	ErrInvalidPath         = errors.New("path is invalid")
	ErrRepositoryPathEmpty = errors.InvalidArgument("repository path cannot be empty")
	ErrBranchNameEmpty     = errors.InvalidArgument("branch name cannot be empty")
	ErrParseDiffHunkHeader = errors.Internal(nil, "failed to parse diff hunk header")
	ErrNoDefaultBranch     = errors.New("no default branch")
)

// ConcatenateError concatenats an error with stderr string
func ConcatenateError(err error, stderr string) error {
	if len(stderr) == 0 {
		return err
	}
	return fmt.Errorf("%w - %s", err, stderr)
}

type runStdError struct {
	err    error
	stderr string
	errMsg string
}

func (r *runStdError) Error() string {
	// the stderr must be in the returned error text, some code only checks `strings.Contains(err.Error(), "git error")`
	if r.errMsg == "" {
		r.errMsg = ConcatenateError(r.err, r.stderr).Error()
	}
	return r.errMsg
}

func (r *runStdError) Unwrap() error {
	return r.err
}

func (r *runStdError) Stderr() string {
	return r.stderr
}

func (r *runStdError) IsExitCode(code int) bool {
	var exitError *exec.ExitError
	if errors.As(r.err, &exitError) {
		return exitError.ExitCode() == code
	}
	return false
}

// ErrPushOutOfDate represents an error if merging fails due to unrelated histories
type ErrPushOutOfDate struct {
	StdOut string
	StdErr string
	Err    error
}

// IsErrPushOutOfDate checks if an error is a ErrPushOutOfDate.
func IsErrPushOutOfDate(err error) bool {
	_, ok := err.(*ErrPushOutOfDate)
	return ok
}

func (err *ErrPushOutOfDate) Error() string {
	return fmt.Sprintf("PushOutOfDate Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

// Unwrap unwraps the underlying error
func (err *ErrPushOutOfDate) Unwrap() error {
	return fmt.Errorf("%v - %s", err.Err, err.StdErr)
}

// ErrPushRejected represents an error if merging fails due to rejection from a hook
type ErrPushRejected struct {
	Message string
	StdOut  string
	StdErr  string
	Err     error
}

// IsErrPushRejected checks if an error is a ErrPushRejected.
func IsErrPushRejected(err error) bool {
	_, ok := err.(*ErrPushRejected)
	return ok
}

func (err *ErrPushRejected) Error() string {
	return fmt.Sprintf("PushRejected Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

// Unwrap unwraps the underlying error
func (err *ErrPushRejected) Unwrap() error {
	return fmt.Errorf("%v - %s", err.Err, err.StdErr)
}

// GenerateMessage generates the remote message from the stderr
func (err *ErrPushRejected) GenerateMessage() {
	messageBuilder := &strings.Builder{}
	i := strings.Index(err.StdErr, "remote: ")
	if i < 0 {
		err.Message = ""
		return
	}
	for {
		if len(err.StdErr) <= i+8 {
			break
		}
		if err.StdErr[i:i+8] != "remote: " {
			break
		}
		i += 8
		nl := strings.IndexByte(err.StdErr[i:], '\n')
		if nl >= 0 {
			messageBuilder.WriteString(err.StdErr[i : i+nl+1])
			i = i + nl + 1
		} else {
			messageBuilder.WriteString(err.StdErr[i:])
			i = len(err.StdErr)
		}
	}
	err.Message = strings.TrimSpace(messageBuilder.String())
}

// ErrMoreThanOne represents an error if pull request fails when there are more than one sources (branch, tag) with the same name
type ErrMoreThanOne struct {
	StdOut string
	StdErr string
	Err    error
}

// IsErrMoreThanOne checks if an error is a ErrMoreThanOne
func IsErrMoreThanOne(err error) bool {
	_, ok := err.(*ErrMoreThanOne)
	return ok
}

func (err *ErrMoreThanOne) Error() string {
	return fmt.Sprintf("ErrMoreThanOne Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

// Logs the error and message, returns either the provided message or a git equivalent if possible.
// Always logs the full message with error as warning.
func processGitErrorf(err error, format string, args ...interface{}) error {
	// create fallback error returned if we can't map it
	fallbackErr := errors.Internal(err, format, args...)

	// always log internal error together with message.
	log.Warn().Msgf("%v: [GIT] %v", fallbackErr, err)

	// check if it's a RunStdError error (contains raw git error)
	var runStdErr *runStdError
	if errors.As(err, &runStdErr) {
		return mapRunStdError(runStdErr, fallbackErr)
	}

	switch {
	case err.Error() == "no such file or directory":
		return errors.NotFound("repository not found")
	default:
		return fallbackErr
	}
}

// Doubt this will work for all std errors, as git doesn't seem to have nice error codes.
func mapRunStdError(err *runStdError, fallback error) error {
	switch {
	// exit status 128 - fatal: A branch named 'mybranch' already exists.
	// exit status 128 - fatal: cannot lock ref 'refs/heads/a': 'refs/heads/a/b' exists; cannot create 'refs/heads/a'
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "exists"):
		return errors.Conflict(err.Stderr())

	// exit status 128 - fatal: 'a/bc/d/' is not a valid branch name.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "not a valid"):
		return errors.InvalidArgument(err.Stderr())

	// exit status 1 - error: branch 'mybranch' not found.
	case err.IsExitCode(1) && strings.Contains(err.Stderr(), "not found"):
		return errors.NotFound(err.Stderr())

	// exit status 128 - fatal: ambiguous argument 'branch1...branch2': unknown revision or path not in the working tree.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "unknown revision"):
		msg := "unknown revision or path not in the working tree"
		// parse the error response from git output
		lines := strings.Split(err.Error(), "\n")
		if len(lines) > 0 {
			cols := strings.Split(lines[0], ": ")
			if len(cols) >= 2 {
				msg = cols[1] + ", " + cols[2]
			}
		}
		return errors.NotFound(msg)

	// exit status 128 - fatal: couldn't find remote ref v1.
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "couldn't find"):
		return errors.NotFound(err.Stderr())

	// exit status 128 - fatal: unable to access 'http://127.0.0.1:4101/hvfl1xj5fojwlrw77xjflw80uxjous254jrr967rvj/':
	//   Failed to connect to 127.0.0.1 port 4101 after 4 ms: Connection refused
	case err.IsExitCode(128) && strings.Contains(err.Stderr(), "Failed to connect"):
		return errors.Internal(err, "failed to connect")

	default:
		return fallback
	}
}

func ErrNotExist(id, relPath string) error {
	return errors.NotFound("object does not exist [id: %s, rel_path: %s]", id, relPath)
}

const (
	StatusNotMergeable errors.Status = "not_mergeable"
)

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
