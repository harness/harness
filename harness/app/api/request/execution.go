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

const QueryParamPipelineIdentifier = "pipeline_identifier"

// ParseSortExecution extracts the execution sort parameter from the url.
func ParseSortExecution(r *http.Request) enum.ExecutionSort {
	result, _ := enum.ExecutionSort(r.URL.Query().Get(QueryParamSort)).Sanitize()
	return result
}

func ParseListExecutionsFilterFromRequest(r *http.Request) (types.ListExecutionsFilter, error) {
	return types.ListExecutionsFilter{
		ListQueryFilter: types.ListQueryFilter{
			Query:      ParseQuery(r),
			Pagination: ParsePaginationFromRequest(r),
		},
		PipelineIdentifier: QueryParamOrDefault(r, QueryParamPipelineIdentifier, ""),
		Sort:               ParseSortExecution(r),
		Order:              ParseOrder(r),
	}, nil
}
