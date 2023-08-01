// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ParseMembershipSort extracts the membership sort parameter from the url.
func ParseMembershipSort(r *http.Request) enum.MembershipSort {
	return enum.ParseMembershipSort(
		r.URL.Query().Get(QueryParamSort),
	)
}

// ParseMembershipFilter extracts the membership filter from the url.
func ParseMembershipFilter(r *http.Request) types.MembershipFilter {
	return types.MembershipFilter{
		Page:  ParsePage(r),
		Size:  ParseLimit(r),
		Query: ParseQuery(r),
		Sort:  ParseMembershipSort(r),
		Order: ParseOrder(r),
	}
}
