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

package blob

import (
	"net/url"
)

type SignURLConfig struct {
	Method          string
	ContentType     string
	Headers         []string
	QueryParameters url.Values
	Insecure        bool
}

type SignURLOption interface {
	Apply(*SignURLConfig)
}

type SignedURLConfigFunc func(opts *SignURLConfig)

func (opts SignedURLConfigFunc) Apply(config *SignURLConfig) {
	opts(config)
}

// SignWithMethod use http method for signing url request.
func SignWithMethod(method string) SignedURLConfigFunc {
	return func(opts *SignURLConfig) {
		opts.Method = method
	}
}

// SignWithContentType use http content type for signing url request.
func SignWithContentType(contentType string) SignedURLConfigFunc {
	return func(opts *SignURLConfig) {
		opts.ContentType = contentType
	}
}

// SignWithHeaders use http headers for signing url request.
func SignWithHeaders(headers []string) SignedURLConfigFunc {
	return func(opts *SignURLConfig) {
		opts.Headers = headers
	}
}

// SignWithQueryParameters use http query params for signing url request.
func SignWithQueryParameters(queryParameters url.Values) SignedURLConfigFunc {
	return func(opts *SignURLConfig) {
		opts.QueryParameters = queryParameters
	}
}

// SignWithInsecure use http insecure (no https) for signing url request.
func SignWithInsecure(insecure bool) SignedURLConfigFunc {
	return func(opts *SignURLConfig) {
		opts.Insecure = insecure
	}
}
