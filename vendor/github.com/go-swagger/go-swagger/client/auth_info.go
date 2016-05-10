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

import "github.com/go-swagger/go-swagger/strfmt"

// AuthInfoWriterFunc converts a function to a request writer interface
type AuthInfoWriterFunc func(Request, strfmt.Registry) error

// AuthenticateRequest adds authentication data to the request
func (fn AuthInfoWriterFunc) AuthenticateRequest(req Request, reg strfmt.Registry) error {
	return fn(req, reg)
}

// An AuthInfoWriter implementor knows how to write authentication info to a request
type AuthInfoWriter interface {
	AuthenticateRequest(Request, strfmt.Registry) error
}
