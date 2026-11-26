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
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestRemainderFromPath(t *testing.T) {
	req := newTestRequest(t, "/", PathParamRemainder, "dir%2Ffile.txt")
	require.Equal(t, "dir/file.txt", GetOptionalRemainderFromPath(req))
	got, err := GetRemainderFromPath(req)
	require.NoError(t, err)
	require.Equal(t, "dir/file.txt", got)

	empty := newTestRequest(t, "/")
	require.Empty(t, GetOptionalRemainderFromPath(empty))
	_, err = GetRemainderFromPath(empty)
	requireBadRequest(t, err, "not found")
}

func TestPagination(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantPage  int
		wantLimit int
	}{
		{"defaults", "", 1, PerPageDefault},
		{"explicit", "page=3&limit=10", 3, 10},
		{"invalid page falls back", "page=abc&limit=10", 1, 10},
		{"negative page falls back", "page=-4", 1, PerPageDefault},
		{"limit is capped", "limit=100000", 1, PerPageMax},
		{"zero limit uses the default", "limit=0", 1, PerPageDefault},
		{"invalid limit uses the default", "limit=x", 1, PerPageDefault},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newTestRequest(t, "/?"+tt.query)
			require.Equal(t, tt.wantPage, ParsePage(req))
			require.Equal(t, tt.wantLimit, ParseLimit(req))
			require.Equal(t, types.Pagination{Page: tt.wantPage, Size: tt.wantLimit}, ParsePaginationFromRequest(req))
		})
	}

	req := newTestRequest(t, "/?limit=7")
	require.Equal(t, uint64(7), ParseLimitOrDefaultWithMax(req, 5, 10))
	require.Equal(t, uint64(5), ParseLimitOrDefaultWithMax(newTestRequest(t, "/"), 5, 10))
	require.Equal(t, uint64(3), ParseLimitOrDefaultWithMax(req, 1, 3))
}

func TestQueryOrderSortFilter(t *testing.T) {
	req := newTestRequest(t, "/?query=needle&sort=name&order=desc&page=2&limit=5")

	require.Equal(t, "needle", ParseQuery(req))
	require.Equal(t, "name", ParseSort(req))
	require.Equal(t, enum.OrderDesc, ParseOrder(req))
	require.Equal(t, enum.OrderDefault, ParseOrder(newTestRequest(t, "/")))

	require.Equal(t, types.ListQueryFilter{
		Query:      "needle",
		Pagination: types.Pagination{Page: 2, Size: 5},
	}, ParseListQueryFilterFromRequest(req))
}

func TestTimeFilters(t *testing.T) {
	req := newTestRequest(t, "/?created_lt=10&created_gt=5&updated_lt=20&updated_gt=15&edited_lt=30&edited_gt=25")

	created, err := ParseCreated(req)
	require.NoError(t, err)
	require.Equal(t, types.CreatedFilter{CreatedLt: 10, CreatedGt: 5}, created)

	updated, err := ParseUpdated(req)
	require.NoError(t, err)
	require.Equal(t, types.UpdatedFilter{UpdatedLt: 20, UpdatedGt: 15}, updated)

	edited, err := ParseEdited(req)
	require.NoError(t, err)
	require.Equal(t, types.EditedFilter{EditedLt: 30, EditedGt: 25}, edited)

	none := newTestRequest(t, "/")
	created, err = ParseCreated(none)
	require.NoError(t, err)
	require.Equal(t, types.CreatedFilter{}, created)

	for _, name := range []string{"created_lt", "created_gt", "updated_lt", "updated_gt", "edited_lt", "edited_gt"} {
		bad := newTestRequest(t, "/?"+name+"=nope")
		_, errCreated := ParseCreated(bad)
		_, errUpdated := ParseUpdated(bad)
		_, errEdited := ParseEdited(bad)
		require.Equal(t, 1, countErrors(errCreated, errUpdated, errEdited), name)
	}
}

func countErrors(errs ...error) int {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	return n
}

func TestBooleanOptions(t *testing.T) {
	type option struct {
		name  string
		param string
		parse func(*testing.T, string) (bool, error)
	}
	options := []option{
		{"recursive", QueryParamRecursive, nil},
		{"regex", QueryParamRegex, nil},
		{"inherited", QueryParamInherited, nil},
		{"include pullreq count", QueryParamIncludePullreqCount, nil},
		{"include values", QueryParamIncludeValues, nil},
		{"assignable", QueryParamAssignable, nil},
	}
	parsers := map[string]func(*testing.T, string) (bool, error){
		QueryParamRecursive: func(t *testing.T, q string) (bool, error) {
			return ParseRecursiveFromQuery(newTestRequest(t, "/?"+q))
		},
		QueryParamRegex: func(t *testing.T, q string) (bool, error) {
			return ParseRegexFromQuery(newTestRequest(t, "/?"+q))
		},
		QueryParamInherited: func(t *testing.T, q string) (bool, error) {
			return ParseInheritedFromQuery(newTestRequest(t, "/?"+q))
		},
		QueryParamIncludePullreqCount: func(t *testing.T, q string) (bool, error) {
			return ParseIncludePullreqCountFromQuery(newTestRequest(t, "/?"+q))
		},
		QueryParamIncludeValues: func(t *testing.T, q string) (bool, error) {
			return ParseIncludeValuesFromQuery(newTestRequest(t, "/?"+q))
		},
		QueryParamAssignable: func(t *testing.T, q string) (bool, error) {
			return ParseAssignableFromQuery(newTestRequest(t, "/?"+q))
		},
	}
	for _, o := range options {
		t.Run(o.name, func(t *testing.T) {
			parse := parsers[o.param]

			got, err := parse(t, o.param+"=true")
			require.NoError(t, err)
			require.True(t, got)

			got, err = parse(t, "")
			require.NoError(t, err)
			require.False(t, got, "defaults to false")

			_, err = parse(t, o.param+"=maybe")
			requireBadRequest(t, err, "must be a boolean")
		})
	}
}

func TestDeletedFilters(t *testing.T) {
	req := newTestRequest(t, "/?deleted_at=100&deleted_before_or_at=200")

	at, err := GetDeletedAtFromQueryOrError(req)
	require.NoError(t, err)
	require.Equal(t, int64(100), at)
	_, err = GetDeletedAtFromQueryOrError(newTestRequest(t, "/"))
	requireBadRequest(t, err, "not found")

	before, ok, err := GetDeletedBeforeOrAtFromQuery(req)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(200), before)
	_, ok, err = GetDeletedBeforeOrAtFromQuery(newTestRequest(t, "/"))
	require.NoError(t, err)
	require.False(t, ok)

	at, ok, err = GetDeletedAtFromQuery(req)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(100), at)
	_, _, err = GetDeletedAtFromQuery(newTestRequest(t, "/?deleted_at=-5"))
	requireBadRequest(t, err, "must be a positive integer")
}

func TestMembershipFilters(t *testing.T) {
	req := newTestRequest(t, "/?query=jane&sort=created&order=asc&page=2&limit=5")

	require.Equal(t, enum.MembershipUserSortCreated, ParseMembershipUserSort(req))
	require.Equal(t, enum.MembershipUserSortName, ParseMembershipUserSort(newTestRequest(t, "/")))
	user := ParseMembershipUserFilter(req)
	require.Equal(t, "jane", user.Query)
	require.Equal(t, enum.MembershipUserSortCreated, user.Sort)
	require.Equal(t, enum.OrderAsc, user.Order)
	require.Equal(t, types.Pagination{Page: 2, Size: 5}, user.Pagination)

	require.Equal(t, enum.MembershipSpaceSortCreated, ParseMembershipSpaceSort(req))
	require.Equal(t, enum.MembershipSpaceSortIdentifier, ParseMembershipSpaceSort(newTestRequest(t, "/")))
	space := ParseMembershipSpaceFilter(req)
	require.Equal(t, enum.MembershipSpaceSortCreated, space.Sort)
	require.Equal(t, enum.OrderAsc, space.Order)
	require.Equal(t, "jane", space.Query)
}
