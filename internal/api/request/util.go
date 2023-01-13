// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi"
)

const (
	PathParamRemainder = "*"

	QueryParamSort  = "sort"
	QueryParamOrder = "order"
	QueryParamQuery = "query"

	QueryParamCreatedBy = "created_by"

	QueryParamState = "state"
	QueryParamKind  = "kind"
	QueryParamType  = "type"

	QueryParamAfter  = "after"
	QueryParamBefore = "before"

	QueryParamPage  = "page"
	QueryParamLimit = "limit"
	PerPageDefault  = 30
	PerPageMax      = 100
)

// PathParamOrError tries to retrieve the parameter from the request and
// returns the parameter if it exists and is not empty, otherwise returns an error.
func PathParamOrError(r *http.Request, paramName string) (string, error) {
	value := chi.URLParam(r, paramName)
	if value == "" {
		return "", usererror.BadRequest(fmt.Sprintf("Parameter '%s' not found in request path.", paramName))
	}

	return value, nil
}

// PathParamOrEmpty retrieves the path parameter or returns an empty string otherwise.
func PathParamOrEmpty(r *http.Request, paramName string) string {
	return chi.URLParam(r, paramName)
}

// QueryParam returns the parameter if it exists.
func QueryParam(r *http.Request, paramName string) (string, bool) {
	query := r.URL.Query()
	if !query.Has(paramName) {
		return "", false
	}

	return query.Get(paramName), true
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
		return "", usererror.BadRequest(fmt.Sprintf("Parameter '%s' not found in query.", paramName))
	}

	return val, nil
}

// QueryParamAsID extracts an ID parameter from the url.
func QueryParamAsID(r *http.Request, paramName string) (int64, error) {
	return QueryParamAsPositiveInt64(r, paramName)
}

// QueryParamAsInt64 extracts an integer parameter from the url.
func QueryParamAsInt64(r *http.Request, paramName string) (int64, error) {
	value, ok := QueryParam(r, paramName)
	if !ok {
		return 0, nil
	}

	valueInt, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, usererror.BadRequest(fmt.Sprintf("Parameter '%s' must be an integer.", paramName))
	}

	return valueInt, nil
}

// QueryParamAsPositiveInt64 extracts an integer parameter from the url.
func QueryParamAsPositiveInt64(r *http.Request, paramName string) (int64, error) {
	value, ok := QueryParam(r, paramName)
	if !ok {
		return 0, nil
	}

	valueInt, err := strconv.ParseInt(value, 10, 64)
	if err != nil || valueInt <= 0 {
		return 0, usererror.BadRequest(fmt.Sprintf("Parameter '%s' must be a positive integer.", paramName))
	}

	return valueInt, nil
}

// PathParamAsInt64 tries to retrieve the parameter from the request and parse it to int64.
func PathParamAsInt64(r *http.Request, paramName string) (int64, error) {
	rawValue, err := PathParamOrError(r, paramName)
	if err != nil {
		return 0, err
	}

	valueAsInt64, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse value '%s' of parameter '%s' to int64: %w", rawValue, paramName, err)
	}

	return valueAsInt64, nil
}

// QueryParamAsBoolOrDefault tries to retrieve the parameter from the query and parse it to bool.
func QueryParamAsBoolOrDefault(r *http.Request, paramName string, deflt bool) (bool, error) {
	rawValue, ok := QueryParam(r, paramName)
	if !ok || len(rawValue) == 0 {
		return deflt, nil
	}

	boolValue, err := strconv.ParseBool(rawValue)
	if err != nil {
		return false, fmt.Errorf("failed to parse value '%s' of parameter '%s' to bool: %w", rawValue, paramName, err)
	}

	return boolValue, nil
}

// GetOptionalRemainderFromPath returns the remainder ("*") from the path or an empty string if it doesn't exist.
func GetOptionalRemainderFromPath(r *http.Request) string {
	return PathParamOrEmpty(r, PathParamRemainder)
}

// GetRemainderFromPath returns the remainder ("*") from the path or an an error if it doesn't exist.
func GetRemainderFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamRemainder)
}

// ParseQuery extracts the query parameter from the url.
func ParseQuery(r *http.Request) string {
	return r.FormValue(QueryParamQuery)
}

// ParsePage extracts the page parameter from the url.
func ParsePage(r *http.Request) int {
	s := r.FormValue(QueryParamPage)
	i, _ := strconv.Atoi(s)
	if i <= 0 {
		i = 1
	}
	return i
}

// ParseLimit extracts the limit parameter from the url.
func ParseLimit(r *http.Request) int {
	s := r.FormValue(QueryParamLimit)
	i, _ := strconv.Atoi(s)
	if i <= 0 {
		i = PerPageDefault
	} else if i > PerPageMax {
		i = PerPageMax
	}
	return i
}

// ParseOrder extracts the order parameter from the url.
func ParseOrder(r *http.Request) enum.Order {
	return enum.ParseOrder(
		r.FormValue(QueryParamOrder),
	)
}

// ParseSort extracts the sort parameter from the url.
func ParseSort(r *http.Request) string {
	return r.FormValue(QueryParamSort)
}
