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

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PathParamPullReqNumber    = "pullreq_number"
	PathParamPullReqCommentID = "pullreq_comment_id"
	PathParamReviewerID       = "pullreq_reviewer_id"
	PathParamUserGroupID      = "user_group_id"
	PathParamSourceBranch     = "source_branch"
	PathParamTargetBranch     = "target_branch"

	QueryParamCommenterID        = "commenter_id"
	QueryParamReviewerID         = "reviewer_id"
	QueryParamReviewDecision     = "review_decision"
	QueryParamMentionedID        = "mentioned_id"
	QueryParamExcludeDescription = "exclude_description"
	QueryParamSourceRepoRef      = "source_repo_ref"
	QueryParamSourceBranch       = "source_branch"
	QueryParamTargetBranch       = "target_branch"
)

func GetPullReqNumberFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamPullReqNumber)
}

func GetReviewerIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamReviewerID)
}
func GetUserGroupIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamUserGroupID)
}

func GetPullReqCommentIDPath(r *http.Request) (int64, error) {
	return PathParamAsPositiveInt64(r, PathParamPullReqCommentID)
}

func GetPullReqSourceBranchFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamSourceBranch)
}

func GetPullReqTargetBranchFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamTargetBranch)
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

// parseReviewDecisions extracts the pull request reviewer decisions from the url.
func parseReviewDecisions(r *http.Request) []enum.PullReqReviewDecision {
	strReviewDecisions, _ := QueryParamList(r, QueryParamReviewDecision)
	m := make(map[enum.PullReqReviewDecision]struct{}) // use map to eliminate duplicates
	for _, s := range strReviewDecisions {
		if state, ok := enum.PullReqReviewDecision(s).Sanitize(); ok {
			m[state] = struct{}{}
		}
	}

	reviewDecisions := make([]enum.PullReqReviewDecision, 0, len(m))
	for s := range m {
		reviewDecisions = append(reviewDecisions, s)
	}

	return reviewDecisions
}

func ParsePullReqMetadataOptions(r *http.Request) (types.PullReqMetadataOptions, error) {
	// TODO: Remove the "includeGitStats := true" line and uncomment the following code block.
	// Because introduction of "include_git_stats" parameter is a breaking API change,
	// we should remove this line only after other teams, who need PR stats
	// include the include_git_stats=true in API calls.
	includeGitStats := true
	/*includeGitStats, err := GetIncludeGitStatsFromQueryOrDefault(r, false)
	if err != nil {
		return types.PullReqMetadataOptions{}, err
	}*/

	includeChecks, err := GetIncludeChecksFromQueryOrDefault(r, false)
	if err != nil {
		return types.PullReqMetadataOptions{}, err
	}

	includeRules, err := GetIncludeRulesFromQueryOrDefault(r, false)
	if err != nil {
		return types.PullReqMetadataOptions{}, err
	}

	return types.PullReqMetadataOptions{
		IncludeGitStats: includeGitStats,
		IncludeChecks:   includeChecks,
		IncludeRules:    includeRules,
	}, nil
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

	createdFilter, err := ParseCreated(r)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing pr created filter: %w", err)
	}

	updatedFilter, err := ParseUpdated(r)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing pr updated filter: %w", err)
	}

	editedFilter, err := ParseEdited(r)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing pr edited filter: %w", err)
	}

	excludeDescription, err := QueryParamAsBoolOrDefault(r, QueryParamExcludeDescription, false)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing include description filter: %w", err)
	}

	authorID, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamAuthorID, 0)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing author ID filter: %w", err)
	}

	commenterID, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamCommenterID, 0)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing commenter ID filter: %w", err)
	}

	reviewerID, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamReviewerID, 0)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing reviewer ID filter: %w", err)
	}

	reviewDecisions := parseReviewDecisions(r)
	if len(reviewDecisions) > 0 && reviewerID <= 0 {
		return nil, errors.InvalidArgument("Can't use review decisions without providing a reviewer ID")
	}

	mentionedID, err := QueryParamAsPositiveInt64OrDefault(r, QueryParamMentionedID, 0)
	if err != nil {
		return nil, fmt.Errorf("encountered error parsing mentioned ID filter: %w", err)
	}

	metadataOptions, err := ParsePullReqMetadataOptions(r)
	if err != nil {
		return nil, err
	}

	if authorID > 0 {
		createdBy = append(createdBy, authorID)
	}

	return &types.PullReqFilter{
		Page:                   ParsePage(r),
		Size:                   ParseLimit(r),
		Query:                  ParseQuery(r),
		CreatedBy:              createdBy,
		SourceRepoRef:          r.URL.Query().Get(QueryParamSourceRepoRef),
		SourceBranch:           r.URL.Query().Get(QueryParamSourceBranch),
		TargetBranch:           r.URL.Query().Get(QueryParamTargetBranch),
		States:                 parsePullReqStates(r),
		Sort:                   ParseSortPullReq(r),
		Order:                  ParseOrder(r),
		LabelID:                labelID,
		ValueID:                valueID,
		CommenterID:            commenterID,
		ReviewerID:             reviewerID,
		ReviewDecisions:        reviewDecisions,
		MentionedID:            mentionedID,
		ExcludeDescription:     excludeDescription,
		CreatedFilter:          createdFilter,
		UpdatedFilter:          updatedFilter,
		EditedFilter:           editedFilter,
		PullReqMetadataOptions: metadataOptions,
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
