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

// PullReqReviewerSuggestion represents a suggested reviewer for a pull request.
type PullReqReviewerSuggestion struct {
	PullReqID   int64 `json:"pullreq_id"`
	CreatedBy   int64 `json:"created_by"`
	PrincipalID int64 `json:"principal_id"`
	Created     int64 `json:"created"`
}

// PullReqReviewerSuggestionInfo is the API response item for a reviewer suggestion.
type PullReqReviewerSuggestionInfo struct {
	Reviewer    PrincipalInfo `json:"reviewer"`
	SuggestedBy PrincipalInfo `json:"suggested_by"`
	SuggestedAt int64         `json:"suggested_at"`
}

// ListReviewerSuggestionsOutput is the API response for listing reviewer suggestions.
type ListReviewerSuggestionsOutput struct {
	Suggestions []*PullReqReviewerSuggestionInfo `json:"suggestions"`
}
