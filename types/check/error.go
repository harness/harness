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
