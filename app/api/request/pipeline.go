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
)

const (
	PathParamPipelineIdentifier = "pipeline_identifier"
	PathParamExecutionNumber    = "execution_number"
	PathParamLastExecutions     = "last_executions"
	PathParamStageNumber        = "stage_number"
	PathParamStepNumber         = "step_number"
	PathParamTriggerIdentifier  = "trigger_identifier"
	QueryParamLatest            = "latest"
	QueryParamBranch            = "branch"
)

func GetPipelineIdentifierFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamPipelineIdentifier)
}

func GetBranchFromQuery(r *http.Request) string {
	return QueryParamOrDefault(r, QueryParamBranch, "")
}

func GetExecutionNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamExecutionNumber)
}

func GetStageNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamStageNumber)
}

func GetStepNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamStepNumber)
}

func GetLatestFromPath(r *http.Request) bool {
	v, _ := QueryParam(r, QueryParamLatest)
	return v == "true"
}

func GetTriggerIdentifierFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamTriggerIdentifier)
}

func ParseListPipelinesFilterFromRequest(r *http.Request) (types.ListPipelinesFilter, error) {
	lastExecs, err := QueryParamAsPositiveInt64OrDefault(r, PathParamLastExecutions, 10)
	if err != nil {
		return types.ListPipelinesFilter{}, err
	}

	return types.ListPipelinesFilter{
		ListQueryFilter: types.ListQueryFilter{
			Query:      ParseQuery(r),
			Pagination: ParsePaginationFromRequest(r),
		},
		Latest:         GetLatestFromPath(r),
		LastExecutions: lastExecs,
	}, nil
}
