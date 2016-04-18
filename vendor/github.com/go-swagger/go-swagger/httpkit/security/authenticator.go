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

package security

import (
	"net/http"
	"strings"

	"github.com/go-swagger/go-swagger/errors"
	"github.com/go-swagger/go-swagger/httpkit"
)

// httpAuthenticator is a function that authenticates a HTTP request
func httpAuthenticator(handler func(*http.Request) (bool, interface{}, error)) httpkit.Authenticator {
	return httpkit.AuthenticatorFunc(func(params interface{}) (bool, interface{}, error) {
		if request, ok := params.(*http.Request); ok {
			return handler(request)
		}
		return false, nil, nil
	})
}

// UserPassAuthentication authentication function
type UserPassAuthentication func(string, string) (interface{}, error)

// TokenAuthentication authentication function
type TokenAuthentication func(string) (interface{}, error)

// BasicAuth creates a basic auth authenticator with the provided authentication function
func BasicAuth(authenticate UserPassAuthentication) httpkit.Authenticator {
	return httpAuthenticator(func(r *http.Request) (bool, interface{}, error) {
		if usr, pass, ok := r.BasicAuth(); ok {
			p, err := authenticate(usr, pass)
			return true, p, err
		}

		return false, nil, nil
	})
}

// APIKeyAuth creates an authenticator that uses a token for authorization.
// This token can be obtained from either a header or a query string
func APIKeyAuth(name, in string, authenticate TokenAuthentication) httpkit.Authenticator {
	inl := strings.ToLower(in)
	if inl != "query" && inl != "header" {
		// panic because this is most likely a typo
		panic(errors.New(500, "api key auth: in value needs to be either \"query\" or \"header\"."))
	}

	var getToken func(*http.Request) string
	switch inl {
	case "header":
		getToken = func(r *http.Request) string { return r.Header.Get(name) }
	case "query":
		getToken = func(r *http.Request) string { return r.URL.Query().Get(name) }
	}

	return httpAuthenticator(func(r *http.Request) (bool, interface{}, error) {
		token := getToken(r)
		if token == "" {
			return false, nil, nil
		}

		p, err := authenticate(token)
		return true, p, err
	})
}
