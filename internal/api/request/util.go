// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
	"strconv"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

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
	s := r.FormValue("per_page")
	i, _ := strconv.Atoi(s)
	if i == 0 {
		i = 100
	} else if i > 100 {
		i = 100
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
func ParseSort(r *http.Request) (s string) {
	return r.FormValue("sort")
}

// ParseSortUser extracts the user stor parameter from the url.
func ParseSortUser(r *http.Request) enum.UserAttr {
	return enum.ParseUserAttr(
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
func ParseUserFilter(r *http.Request) types.UserFilter {
	return types.UserFilter{
		Order: ParseOrder(r),
		Page:  ParsePage(r),
		Sort:  ParseSortUser(r),
		Size:  ParseSize(r),
	}
}
