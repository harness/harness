// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"net/url"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamSpaceRef = "space_ref"
)

func GetSpaceRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamSpaceRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped and lower
	return url.PathUnescape(rawRef)
}

// ParseSortSpace extracts the space sort parameter from the url.
func ParseSortSpace(r *http.Request) enum.SpaceAttr {
	return enum.ParseSpaceAttr(
		r.FormValue(QueryParamSort),
	)
}

// ParseSpaceFilter extracts the space filter from the url.
func ParseSpaceFilter(r *http.Request) *types.SpaceFilter {
	return &types.SpaceFilter{
		Query: ParseQuery(r),
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortSpace(r),
		Size:  ParseLimit(r),
	}
}
