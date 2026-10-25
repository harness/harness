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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamTaskNumber = "task_number"

	QueryParamTaskPriority        = "priority"
	QueryParamTaskOrigin          = "origin"
	QueryParamTaskAssigneeID      = "assignee_id"
	QueryParamTaskAssigneeGroupID = "assignee_group_id"
	QueryParamTaskIsAgent         = "is_agent"
)

// GetTaskNumberFromPath extracts the task number from the url path.
func GetTaskNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamTaskNumber)
}

// ParseTaskFilter extracts the task list filter from the url query.
// Unknown enum values are rejected with a bad request.
func ParseTaskFilter(r *http.Request) (*types.TaskFilter, error) {
	filter := &types.TaskFilter{
		Page:  ParsePage(r),
		Size:  ParseLimit(r),
		Query: ParseQuery(r),
	}

	states, _ := QueryParamList(r, QueryParamState)
	for _, s := range states {
		v, ok := enum.TaskState(s).Sanitize()
		if !ok {
			return nil, usererror.BadRequestf("Unknown task state %q.", s)
		}
		filter.States = appendUnique(filter.States, v)
	}

	priorities, _ := QueryParamList(r, QueryParamTaskPriority)
	for _, s := range priorities {
		v, ok := enum.TaskPriority(s).Sanitize()
		if !ok {
			return nil, usererror.BadRequestf("Unknown task priority %q.", s)
		}
		filter.Priorities = appendUnique(filter.Priorities, v)
	}

	origins, _ := QueryParamList(r, QueryParamTaskOrigin)
	for _, s := range origins {
		v, ok := enum.TaskOrigin(s).Sanitize()
		if !ok {
			return nil, usererror.BadRequestf("Unknown task origin %q.", s)
		}
		filter.Origins = appendUnique(filter.Origins, v)
	}

	var err error
	if filter.AssigneeIDs, err = QueryParamListAsPositiveInt64(r, QueryParamTaskAssigneeID); err != nil {
		return nil, err
	}
	if filter.LabelIDs, err = QueryParamListAsPositiveInt64(r, QueryParamLabelID); err != nil {
		return nil, err
	}

	if groupID, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamTaskAssigneeGroupID, 0); err != nil {
		return nil, err
	} else if groupID > 0 {
		filter.AssigneeGroupID = &groupID
	}

	if _, ok := QueryParam(r, QueryParamTaskIsAgent); ok {
		isAgent, err := QueryParamAsBoolOrDefault(r, QueryParamTaskIsAgent, false)
		if err != nil {
			return nil, err
		}
		filter.IsAgent = &isAgent
	}

	return filter, nil
}

func appendUnique[T comparable](list []T, v T) []T {
	for _, e := range list {
		if e == v {
			return list
		}
	}
	return append(list, v)
}
