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
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamPrincipalUID      = "principal_uid"
	PathParamUserUID           = "user_uid"
	PathParamUserID            = "user_id"
	PathParamServiceAccountUID = "sa_uid"

	PathParamPrincipalID = "principal_id"
)

// GetUserIDFromPath returns the user id from the request path.
func GetUserIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamUserID)
}

// GetPrincipalIDFromPath returns the user id from the request path.
func GetPrincipalIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamPrincipalID)
}

func GetPrincipalUIDFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamPrincipalUID)
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
		r.URL.Query().Get(QueryParamSort),
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
	pTypesRaw := r.URL.Query()[QueryParamType]
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
