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
	PathParamPathID = "path_id"
)

func GetPathIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsInt64(r, PathParamPathID)
}

// ParseSortPath extracts the path sort parameter from the url.
func ParseSortPath(r *http.Request) enum.PathAttr {
	return enum.ParsePathAttr(
		r.FormValue(QueryParamSort),
	)
}

// ParsePathFilter extracts the path filter from the url.
func ParsePathFilter(r *http.Request) *types.PathFilter {
	return &types.PathFilter{
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortPath(r),
		Size:  ParseLimit(r),
	}
}
