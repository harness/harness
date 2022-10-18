// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamRemainder = "*"
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
	if !ok {
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

// ParsePage extracts the page parameter from the url.
func ParsePage(r *http.Request) int {
	s := r.FormValue("page")
	i, _ := strconv.Atoi(s)
	if i == 0 {
		i = 1
	}
	return i
}

// ParseSize extracts the size parameter from the url.
func ParseSize(r *http.Request) int {
	const itemsPerPage = 100
	s := r.FormValue("per_page")
	i, _ := strconv.Atoi(s)
	if i == 0 {
		i = itemsPerPage
	} else if i > itemsPerPage {
		i = itemsPerPage
	}
	return i
}

// ParseOrder extracts the order parameter from the url.
func ParseOrder(r *http.Request) enum.Order {
	return enum.ParseOrder(
		r.FormValue("direction"),
	)
}

// ParseSort extracts the sort parameter from the url.
func ParseSort(r *http.Request) string {
	return r.FormValue("sort")
}

// ParseSortUser extracts the user sort parameter from the url.
func ParseSortUser(r *http.Request) enum.UserAttr {
	return enum.ParseUserAttr(
		r.FormValue("sort"),
	)
}

// ParseSortSpace extracts the space sort parameter from the url.
func ParseSortSpace(r *http.Request) enum.SpaceAttr {
	return enum.ParseSpaceAttr(
		r.FormValue("sort"),
	)
}

// ParseSortRepo extracts the repo sort parameter from the url.
func ParseSortRepo(r *http.Request) enum.RepoAttr {
	return enum.ParseRepoAtrr(
		r.FormValue("sort"),
	)
}

// ParseSortPath extracts the path sort parameter from the url.
func ParseSortPath(r *http.Request) enum.PathAttr {
	return enum.ParsePathAttr(
		r.FormValue("sort"),
	)
}

// ParseParams extracts the query parameter from the url.
func ParseParams(r *http.Request) types.Params {
	return types.Params{
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSort(r),
		Size:  ParseSize(r),
	}
}

// ParseUserFilter extracts the user query parameter from the url.
func ParseUserFilter(r *http.Request) *types.UserFilter {
	return &types.UserFilter{
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortUser(r),
		Size:  ParseSize(r),
	}
}

// ParseSpaceFilter extracts the space query parameter from the url.
func ParseSpaceFilter(r *http.Request) *types.SpaceFilter {
	return &types.SpaceFilter{
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortSpace(r),
		Size:  ParseSize(r),
	}
}

// ParseRepoFilter extracts the repository query parameter from the url.
func ParseRepoFilter(r *http.Request) *types.RepoFilter {
	return &types.RepoFilter{
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortRepo(r),
		Size:  ParseSize(r),
	}
}

// ParsePathFilter extracts the path query parameter from the url.
func ParsePathFilter(r *http.Request) *types.PathFilter {
	return &types.PathFilter{
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortPath(r),
		Size:  ParseSize(r),
	}
}

// ParseCommitFilter extracts the commit query parameter from the url.
// TODO: do we need a separate filter?
func ParseCommitFilter(r *http.Request) *types.CommitFilter {
	return &types.CommitFilter{
		Page: ParsePage(r),
		Size: ParseSize(r),
	}
}

// ParseBranchFilter extracts the branch query parameter from the url.
// TODO: do we need a separate filter?
func ParseBranchFilter(r *http.Request) *types.BranchFilter {
	return &types.BranchFilter{
		Page: ParsePage(r),
		Size: ParseSize(r),
	}
}
