// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamPullReqNumber    = "pullreq_number"
	PathParamPullReqCommentID = "pullreq_comment_id"
)

func GetPullReqNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsInt64(r, PathParamPullReqNumber)
}

func GetPullReqCommentIDPath(r *http.Request) (int64, error) {
	return PathParamAsInt64(r, PathParamPullReqCommentID)
}

// ParseSortPullReq extracts the pull request sort parameter from the url.
func ParseSortPullReq(r *http.Request) enum.PullReqSort {
	result, _ := enum.ParsePullReqSort(enum.PullReqSort(r.FormValue(QueryParamSort)))
	return result
}

// parsePullReqStates extracts the pull request states from the url.
func parsePullReqStates(r *http.Request) []enum.PullReqState {
	strStates := r.Form[QueryParamState]
	m := make(map[enum.PullReqState]struct{}) // use map to eliminate duplicates
	for _, s := range strStates {
		if state, ok := enum.ParsePullReqState(enum.PullReqState(s)); ok {
			m[state] = struct{}{}
		}
	}

	if len(m) == 0 {
		return enum.GetAllPullReqStates() // the default is all PRs
	}

	states := make([]enum.PullReqState, 0, len(m))
	for s := range m {
		states = append(states, s)
	}

	return states
}

// ParsePullReqFilter extracts the pull request query parameter from the url.
func ParsePullReqFilter(r *http.Request) (*types.PullReqFilter, error) {
	createdBy, err := QueryParamAsID(r, QueryParamCreatedBy)
	if err != nil {
		return nil, err
	}
	return &types.PullReqFilter{
		Page:          ParsePage(r),
		Size:          ParseLimit(r),
		Query:         ParseQuery(r),
		CreatedBy:     createdBy,
		SourceRepoRef: r.FormValue("source_repo_ref"),
		SourceBranch:  r.FormValue("source_branch"),
		TargetBranch:  r.FormValue("target_branch"),
		States:        parsePullReqStates(r),
		Sort:          ParseSortPullReq(r),
		Order:         ParseOrder(r),
	}, nil
}

// ParsePullReqActivityFilter extracts the pull request activity query parameter from the url.
func ParsePullReqActivityFilter(r *http.Request) (*types.PullReqActivityFilter, error) {
	after, err := QueryParamAsPositiveInt64(r, QueryParamAfter)
	if err != nil {
		return nil, err
	}
	before, err := QueryParamAsPositiveInt64(r, QueryParamBefore)
	if err != nil {
		return nil, err
	}
	limit, err := QueryParamAsPositiveInt64(r, QueryParamLimit)
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
	strKinds := r.Form[QueryParamKind]
	m := make(map[enum.PullReqActivityKind]struct{}) // use map to eliminate duplicates
	for _, s := range strKinds {
		kind, ok := enum.ParsePullReqActivityKind(enum.PullReqActivityKind(s))
		if !ok {
			continue
		}
		m[kind] = struct{}{}
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
	strType := r.Form[QueryParamType]
	m := make(map[enum.PullReqActivityType]struct{}) // use map to eliminate duplicates
	for _, s := range strType {
		t, ok := enum.ParsePullReqActivityType(enum.PullReqActivityType(s))
		if !ok {
			continue
		}
		m[t] = struct{}{}
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
