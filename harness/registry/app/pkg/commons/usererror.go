//  Copyright 2023 Harness, Inc.
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

package commons

import "net/http"

var (

	// ErrBadRequest is returned when there was an issue with the input.
	ErrBadRequest   = New(http.StatusNotFound, "Bad Request", nil)
	ErrNotSupported = New(http.StatusMethodNotAllowed, "not supported", nil)
)

// Error represents a json-encoded API error.
type Error struct {
	Status  int         `json:"-"`
	Message string      `json:"message"`
	Detail  interface{} `json:"detail,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// New returns a new user facing error.
func New(status int, message string, detail interface{}) *Error {
	return &Error{Status: status, Message: message, Detail: detail}
}

// NotFoundError returns a new user facing not found error.
func NotFoundError(message string, detail interface{}) *Error {
	return New(http.StatusNotFound, message, detail)
}
