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
