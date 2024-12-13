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

package command

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	// ErrInvalidArg represent family of errors to report about bad argument used to make a call.
	ErrInvalidArg = errors.New("invalid argument")
)

// Error type with optional ExitCode and Stderr payload.
type Error struct {
	Err    error
	StdErr []byte
}

// NewError creates error with source err and stderr payload.
func NewError(err error, stderr []byte) *Error {
	return &Error{
		Err:    err,
		StdErr: stderr,
	}
}

func (e *Error) ExitCode() int {
	var exitErr *exec.ExitError
	ok := errors.As(e.Err, &exitErr)
	if ok {
		return exitErr.ExitCode()
	}
	return 0
}

func (e *Error) IsExitCode(code int) bool {
	return e.ExitCode() == code
}

func (e *Error) IsAmbiguousArgErr() bool {
	return strings.Contains(e.Error(), "ambiguous argument")
}

func (e *Error) IsInvalidRefErr() bool {
	return strings.Contains(e.Error(), "not a valid ref")
}

func (e *Error) Error() string {
	if len(e.StdErr) != 0 {
		return fmt.Sprintf("%s: %s", e.Err.Error(), e.StdErr)
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

// AsError unwraps Error otherwise return nil.
func AsError(err error) (e *Error) {
	if errors.As(err, &e) {
		return
	}
	return nil
}
