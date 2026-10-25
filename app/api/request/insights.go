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
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamInsightsProjectID = "project_id"
	PathParamInsightsIssueID   = "issue_id"

	QueryParamInsightsProject = "project"
	QueryParamInsightsStatus  = "status"
	QueryParamInsightsLevel   = "level"
	QueryParamInsightsRelease = "release"
)

// GetInsightsProjectIDFromPath returns the Insights project id from the path.
func GetInsightsProjectIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamInsightsProjectID)
}

// GetInsightsIssueIDFromPath returns the Insights issue id from the path.
func GetInsightsIssueIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamInsightsIssueID)
}

// ParseInsightsIssueFilter extracts the Insights issue filter from the url.
// The status parameter takes a comma separated list of statuses.
func ParseInsightsIssueFilter(r *http.Request) (*types.InsightsIssueFilter, error) {
	q := r.URL.Query()

	f := &types.InsightsIssueFilter{
		ListQueryFilter: ParseListQueryFilterFromRequest(r),
		Level:           q.Get(QueryParamInsightsLevel),
		Release:         q.Get(QueryParamInsightsRelease),
		Sort:            enum.ParseInsightsIssueSort(ParseSort(r)),
		Order:           ParseOrder(r),
	}

	id, _, err := QueryParamAsPositiveInt64(r, QueryParamInsightsProject)
	if err != nil {
		return nil, err
	}
	f.ProjectID = id

	for _, raw := range strings.Split(q.Get(QueryParamInsightsStatus), ",") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		status, ok := enum.InsightsIssueStatus(raw).Sanitize()
		if !ok || string(status) != raw {
			return nil, usererror.BadRequestf("Unknown issue status '%s'.", raw)
		}
		f.Statuses = append(f.Statuses, status)
	}

	return f, nil
}
