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

package types

import "github.com/harness/gitness/types/enum"

// PullReqReviewerSuggestion represents a suggested reviewer for a pull request.
type PullReqReviewerSuggestion struct {
	PullReqID   int64 `json:"pullreq_id"`
	CreatedBy   int64 `json:"created_by"`
	PrincipalID int64 `json:"principal_id"`
	Created     int64 `json:"created"`

	Kind        enum.ReviewerSuggestionKind `json:"kind"`
	Reason      string                      `json:"reason,omitempty"`
	Rank        *int64                      `json:"rank,omitempty"`
	UserGroupID *int64                      `json:"user_group_id,omitempty"`
	CommitSHA   string                      `json:"commit_sha,omitempty"`
}

// PullReqReviewerSuggestionInfo is the API response item for a reviewer suggestion.
type PullReqReviewerSuggestionInfo struct {
	Reviewer    PrincipalInfo `json:"reviewer"`
	SuggestedBy PrincipalInfo `json:"suggested_by"`
	SuggestedAt int64         `json:"suggested_at"`

	Kind      enum.ReviewerSuggestionKind `json:"kind"`
	Reason    string                      `json:"reason,omitempty"`
	Rank      *int64                      `json:"rank,omitempty"`
	UserGroup *UserGroupInfo              `json:"user_group,omitempty"`
	CommitSHA string                      `json:"commit_sha,omitempty"`
}

// ListReviewerSuggestionsOutput is the API response for listing reviewer suggestions.
type ListReviewerSuggestionsOutput struct {
	Suggestions []*PullReqReviewerSuggestionInfo `json:"suggestions"`
}
