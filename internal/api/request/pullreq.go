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
	PathParamPullReqNumber = "pullreqNumber"
)

func GetPullReqNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsInt64(r, PathParamPullReqNumber)
}

// ParseSortPullReq extracts the pull request sort parameter from the url.
func ParseSortPullReq(r *http.Request) enum.PullReqSort {
	return enum.ParsePullReqSort(
		r.FormValue(QueryParamSort),
	)
}

// ParsePullReqStates extracts the pull request states the url.
func ParsePullReqStates(r *http.Request) []enum.PullReqState {
	strStates := r.Form[QueryParamState]
	m := make(map[enum.PullReqState]struct{}) // use map to eliminate duplicates
	for _, s := range strStates {
		state := enum.PullReqState(s)
		if state != enum.PullReqStateOpen && state != enum.PullReqStateMerged &&
			state != enum.PullReqStateClosed && state != enum.PullReqStateRejected {
			continue // skip invalid states
		}
		m[state] = struct{}{}
	}

	if len(m) == 0 {
		return []enum.PullReqState{enum.PullReqStateOpen} // the default is only "open" PRs
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
		Page:      ParsePage(r),
		Size:      ParseSize(r),
		Query:     ParseQuery(r),
		CreatedBy: createdBy,
		States:    ParsePullReqStates(r),
		Sort:      ParseSortPullReq(r),
		Order:     ParseOrder(r),
	}, nil
}
