// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"fmt"
	"io"

	"github.com/go-swagger/go-swagger/httpkit"
)

// A Response represents a client response
// This bridges between responses obtained from different transports
type Response interface {
	Code() int
	Message() string
	GetHeader(string) string
	Body() io.ReadCloser
}

// A ResponseReaderFunc turns a function into a ResponseReader interface implementation
type ResponseReaderFunc func(Response, httpkit.Consumer) (interface{}, error)

// ReadResponse reads the response
func (read ResponseReaderFunc) ReadResponse(resp Response, consumer httpkit.Consumer) (interface{}, error) {
	return read(resp, consumer)
}

// A ResponseReader is an interface for things want to read a response.
// An application of this is to create structs from response values
type ResponseReader interface {
	ReadResponse(Response, httpkit.Consumer) (interface{}, error)
}

// NewAPIError creates a new API error
func NewAPIError(opName string, payload interface{}, code int) *APIError {
	return &APIError{
		OperationName: opName,
		Response:      payload,
		Code:          code,
	}
}

// APIError wraps an error model and captures the status code
type APIError struct {
	OperationName string
	Response      interface{}
	Code          int
}

func (a *APIError) Error() string {
	return fmt.Sprintf("%s (status %d): %+v ", a.OperationName, a.Code, a.Response)
}
