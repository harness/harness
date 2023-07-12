// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamPrincipalUID      = "principal_uid"
	PathParamUserUID           = "user_uid"
	PathParamServiceAccountUID = "sa_uid"

	QueryParamPrincipalID = "principal_id"
)

func GetPrincipalUIDFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamPrincipalUID)
}

// GetPrincipalIDFromQuery returns the principal id from the request query.
func GetPrincipalIDFromQuery(r *http.Request) (int64, error) {
	return QueryParamAsPositiveInt64(r, QueryParamPrincipalID)
}

func GetUserUIDFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamUserUID)
}

func GetServiceAccountUIDFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamServiceAccountUID)
}

// ParseSortUser extracts the user sort parameter from the url.
func ParseSortUser(r *http.Request) enum.UserAttr {
	return enum.ParseUserAttr(
		r.FormValue(QueryParamSort),
	)
}

// ParseUserFilter extracts the user filter from the url.
func ParseUserFilter(r *http.Request) *types.UserFilter {
	return &types.UserFilter{
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortUser(r),
		Size:  ParseLimit(r),
	}
}

// ParsePrincipalTypes extracts the principal types from the url.
func ParsePrincipalTypes(r *http.Request) []enum.PrincipalType {
	pTypesRaw := r.Form[QueryParamType]
	m := make(map[enum.PrincipalType]struct{}) // use map to eliminate duplicates
	for _, pTypeRaw := range pTypesRaw {
		if pType, ok := enum.PrincipalType(pTypeRaw).Sanitize(); ok {
			m[pType] = struct{}{}
		}
	}

	res := make([]enum.PrincipalType, 0, len(m))
	for t := range m {
		res = append(res, t)
	}

	return res
}

// ParsePrincipalFilter extracts the principal filter from the url.
func ParsePrincipalFilter(r *http.Request) *types.PrincipalFilter {
	return &types.PrincipalFilter{
		Query: ParseQuery(r),
		Page:  ParsePage(r),
		Size:  ParseLimit(r),
		Types: ParsePrincipalTypes(r),
	}
}
