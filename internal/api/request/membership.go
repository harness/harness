// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ParseMembershipUserSort extracts the membership sort parameter from the url.
func ParseMembershipUserSort(r *http.Request) enum.MembershipUserSort {
	return enum.ParseMembershipUserSort(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseMembershipUserFilter extracts the membership filter from the url.
func ParseMembershipUserFilter(r *http.Request) types.MembershipUserFilter {
	return types.MembershipUserFilter{
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
		Sort:            ParseMembershipUserSort(r),
		Order:           ParseOrder(r),
	}
}

// ParseMembershipSpaceSort extracts the membership space sort parameter from the url.
func ParseMembershipSpaceSort(r *http.Request) enum.MembershipSpaceSort {
	return enum.ParseMembershipSpaceSort(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseMembershipSpaceFilter extracts the membership space filter from the url.
func ParseMembershipSpaceFilter(r *http.Request) types.MembershipSpaceFilter {
	return types.MembershipSpaceFilter{
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
		Sort:            ParseMembershipSpaceSort(r),
		Order:           ParseOrder(r),
	}
}
