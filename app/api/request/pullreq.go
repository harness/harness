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
	"fmt"
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamPullReqNumber    = "pullreq_number"
	PathParamPullReqCommentID = "pullreq_comment_id"
	PathParamReviewerID       = "pullreq_reviewer_id"
)

func GetPullReqNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamPullReqNumber)
}

func GetReviewerIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamReviewerID)
}

func GetPullReqCommentIDPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamPullReqCommentID)
}

// ParseSortPullReq extracts the pull request sort parameter from the url.
func ParseSortPullReq(r *http.Request) enum.PullReqSort {
	result, _ := enum.PullReqSort(r.URL.Query().Get(QueryParamSort)).Sanitize()
	return result
}

// parsePullReqStates extracts the pull request states from the url.
func parsePullReqStates(r *http.Request) []enum.PullReqState {
	strStates, _ := QueryParamList(r, QueryParamState)
	m := make(map[enum.PullReqState]struct{}) // use map to eliminate duplicates
	for _, s := range strStates {
		if state, ok := enum.PullReqState(s).Sanitize(); ok {
			m[state] = struct{}{}
		}
	}

	states := make([]enum.PullReqState, 0, len(m))
	for s := range m {
		states = append(states, s)
	}

	return states
}

// ParsePullReqFilter extracts the pull request query parameter from the url.
func ParsePullReqFilter(r *http.Request) (*types.PullReqFilter, error) {
	createdBy, err := QueryParamListAsPositiveInt64(r, QueryParamCreatedBy)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing createdby filter: %w", err)
	}

	labelID, err := QueryParamListAsPositiveInt64(r, QueryParamLabelID)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing labelid filter: %w", err)
	}
	valueID, err := QueryParamListAsPositiveInt64(r, QueryParamValueID)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing valueid filter: %w", err)
	}

	createdAtFilter, err := ParseCreated(r)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing pr created filter: %w", err)
	}

	return &types.PullReqFilter{
		Page:          ParsePage(r),
		Size:          ParseLimit(r),
		Query:         ParseQuery(r),
		CreatedBy:     createdBy,
		SourceRepoRef: r.URL.Query().Get("source_repo_ref"),
		SourceBranch:  r.URL.Query().Get("source_branch"),
		TargetBranch:  r.URL.Query().Get("target_branch"),
		States:        parsePullReqStates(r),
		Sort:          ParseSortPullReq(r),
		Order:         ParseOrder(r),
		LabelID:       labelID,
		ValueID:       valueID,
		CreatedFilter: createdAtFilter,
	}, nil
}

// ParsePullReqActivityFilter extracts the pull request activity query parameter from the url.
func ParsePullReqActivityFilter(r *http.Request) (*types.PullReqActivityFilter, error) {
	// after is optional, skipped if set to 0
	after, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamAfter, 0)
	if err != nil {
		return nil, err
	}
	// before is optional, skipped if set to 0
	before, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamBefore, 0)
	if err != nil {
		return nil, err
	}
	// limit is optional, skipped if set to 0
	limit, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamLimit, 0)
	if err != nil {
		return nil, err
	}
	return &types.PullReqActivityFilter{
		After:  after,
		Before: before,
		Limit:  int(limit),
		Types:  parsePullReqActivityTypes(r),
		Kinds:  parsePullReqActivityKinds(r),
	}, nil
}

// parsePullReqActivityKinds extracts the pull request activity kinds from the url.
func parsePullReqActivityKinds(r *http.Request) []enum.PullReqActivityKind {
	strKinds := r.URL.Query()[QueryParamKind]
	m := make(map[enum.PullReqActivityKind]struct{}) // use map to eliminate duplicates
	for _, s := range strKinds {
		if kind, ok := enum.PullReqActivityKind(s).Sanitize(); ok {
			m[kind] = struct{}{}
		}
	}

	if len(m) == 0 {
		return nil
	}

	kinds := make([]enum.PullReqActivityKind, 0, len(m))
	for k := range m {
		kinds = append(kinds, k)
	}

	return kinds
}

// parsePullReqActivityTypes extracts the pull request activity types from the url.
func parsePullReqActivityTypes(r *http.Request) []enum.PullReqActivityType {
	strType := r.URL.Query()[QueryParamType]
	m := make(map[enum.PullReqActivityType]struct{}) // use map to eliminate duplicates
	for _, s := range strType {
		if t, ok := enum.PullReqActivityType(s).Sanitize(); ok {
			m[t] = struct{}{}
		}
	}

	if len(m) == 0 {
		return nil
	}

	activityTypes := make([]enum.PullReqActivityType, 0, len(m))
	for t := range m {
		activityTypes = append(activityTypes, t)
	}

	return activityTypes
}
