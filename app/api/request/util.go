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

package request

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/harness/gitness/app/api/usererror"

	"github.com/go-chi/chi"
)

// GetCookie tries to retrieve the cookie from the request or returns false if it doesn't exist.
func GetCookie(r *http.Request, cookieName string) (string, bool) {
	cookie, err := r.Cookie(cookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return "", false
	} else if err != nil {
		// this should never happen - documentation and code only return `nil` or `http.ErrNoCookie`
		panic(fmt.Sprintf("unexpected error from request.Cookie(...) method: %s", err))
	}

	return cookie.Value, true
}

// GetHeaderOrDefault returns the value of the first non-empty header occurrence.
// If no value is found, the default value is returned.
func GetHeaderOrDefault(r *http.Request, headerName string, dflt string) string {
	val, ok := GetHeader(r, headerName)
	if !ok {
		return dflt
	}

	return val
}

// GetHeader returns the value of the first non-empty header occurrence.
// If no value is found, `false` is returned.
func GetHeader(r *http.Request, headerName string) (string, bool) {
	for _, val := range r.Header.Values(headerName) {
		if val != "" {
			return val, true
		}
	}

	return "", false
}

// PathParamOrError tries to retrieve the parameter from the request and
// returns the parameter if it exists and is not empty, otherwise returns an error.
func PathParamOrError(r *http.Request, paramName string) (string, error) {
	val, ok := PathParam(r, paramName)
	if !ok {
		return "", usererror.BadRequestf("Parameter '%s' not found in request path.", paramName)
	}

	return val, nil
}

// EncodedPathParamOrError tries to retrieve the parameter from the request and
// returns the parameter if it exists and is not empty, otherwise returns an error,
// then it tries to URL decode parameter value,
// and returns decoded value, or an error on decoding failure.
func EncodedPathParamOrError(r *http.Request, paramName string) (string, error) {
	val, err := PathParamOrError(r, paramName)
	if err != nil {
		return "", err
	}

	decoded, err := url.PathUnescape(val)
	if err != nil {
		return "", usererror.BadRequestf("Value %s for param %s has incorrect encoding", val, paramName)
	}

	return decoded, nil
}

// PathParamOrEmpty retrieves the path parameter or returns an empty string otherwise.
func PathParamOrEmpty(r *http.Request, paramName string) string {
	val, ok := PathParam(r, paramName)
	if !ok {
		return ""
	}

	return val
}

// PathParam retrieves the path parameter or returns false if it exists.
func PathParam(r *http.Request, paramName string) (string, bool) {
	val := chi.URLParam(r, paramName)
	if val == "" {
		return "", false
	}

	return val, true
}

// QueryParam returns the parameter if it exists.
func QueryParam(r *http.Request, paramName string) (string, bool) {
	query := r.URL.Query()
	if !query.Has(paramName) {
		return "", false
	}

	return query.Get(paramName), true
}

// QueryParamList returns list of the parameter values if they exist.
func QueryParamList(r *http.Request, paramName string) ([]string, bool) {
	query := r.URL.Query()
	if !query.Has(paramName) {
		return nil, false
	}

	return query[paramName], true
}

// QueryParamOrDefault retrieves the parameter from the query and
// returns the parameter if it exists, otherwise returns the provided default value.
func QueryParamOrDefault(r *http.Request, paramName string, deflt string) string {
	val, ok := QueryParam(r, paramName)
	if !ok {
		return deflt
	}

	return val
}

// QueryParamOrError tries to retrieve the parameter from the query and
// returns the parameter if it exists, otherwise returns an error.
func QueryParamOrError(r *http.Request, paramName string) (string, error) {
	val, ok := QueryParam(r, paramName)
	if !ok {
		return "", usererror.BadRequestf("Parameter '%s' not found in query.", paramName)
	}

	return val, nil
}

// QueryParamAsPositiveInt64 extracts an integer parameter from the request query.
// If the parameter doesn't exist the provided default value is returned.
func QueryParamAsPositiveInt64OrDefault(r *http.Request, paramName string, deflt int64) (int64, error) {
	value, ok := QueryParam(r, paramName)
	if !ok {
		return deflt, nil
	}

	valueInt, err := strconv.ParseInt(value, 10, 64)
	if err != nil || valueInt <= 0 {
		return 0, usererror.BadRequestf("Parameter '%s' must be a positive integer.", paramName)
	}

	return valueInt, nil
}

// QueryParamAsPositiveInt64OrError extracts an integer parameter from the request query.
// If the parameter doesn't exist an error is returned.
func QueryParamAsPositiveInt64OrError(r *http.Request, paramName string) (int64, error) {
	value, err := QueryParamOrError(r, paramName)
	if err != nil {
		return 0, err
	}

	valueInt, err := strconv.ParseInt(value, 10, 64)
	if err != nil || valueInt <= 0 {
		return 0, usererror.BadRequestf("Parameter '%s' must be a positive integer.", paramName)
	}

	return valueInt, nil
}

// QueryParamAsPositiveInt64 extracts an integer parameter from the request query if it exists.
func QueryParamAsPositiveInt64(r *http.Request, paramName string) (int64, bool, error) {
	value, ok := QueryParam(r, paramName)
	if !ok {
		return 0, false, nil
	}

	valueInt, err := strconv.ParseInt(value, 10, 64)
	if err != nil || valueInt <= 0 {
		return 0, false, usererror.BadRequestf("Parameter '%s' must be a positive integer.", paramName)
	}

	return valueInt, true, nil
}

// PathParamAsPositiveInt64 extracts an integer parameter from the request path.
func PathParamAsPositiveInt64(r *http.Request, paramName string) (int64, error) {
	rawValue, err := PathParamOrError(r, paramName)
	if err != nil {
		return 0, err
	}

	valueInt, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil || valueInt <= 0 {
		return 0, usererror.BadRequestf("Parameter '%s' must be a positive integer.", paramName)
	}

	return valueInt, nil
}

// QueryParamAsBoolOrDefault tries to retrieve the parameter from the query and parse it to bool.
func QueryParamAsBoolOrDefault(r *http.Request, paramName string, deflt bool) (bool, error) {
	rawValue, ok := QueryParam(r, paramName)
	if !ok || len(rawValue) == 0 {
		return deflt, nil
	}

	boolValue, err := strconv.ParseBool(rawValue)
	if err != nil {
		return false, usererror.BadRequestf("Parameter '%s' must be a boolean.", paramName)
	}

	return boolValue, nil
}

// QueryParamListAsPositiveInt64 extracts integer parameter slice from the request query.
func QueryParamListAsPositiveInt64(r *http.Request, paramName string) ([]int64, error) {
	valuesString, ok := QueryParamList(r, paramName)
	if !ok {
		return make([]int64, 0), nil
	}
	valuesInt := make([]int64, len(valuesString))

	for i, vs := range valuesString {
		vi, err := strconv.ParseInt(vs, 10, 64)
		if err != nil || vi <= 0 {
			return nil, usererror.BadRequestf("Parameter %q must be a positive integer.", paramName)
		}
		valuesInt[i] = vi
	}
	return valuesInt, nil
}
