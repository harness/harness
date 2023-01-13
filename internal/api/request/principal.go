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
	PathParamUserUID           = "user_uid"
	PathParamServiceAccountUID = "sa_uid"
)

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
