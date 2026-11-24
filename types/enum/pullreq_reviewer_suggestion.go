// Copyright 2026 Harness, Inc.
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

package enum

// ReviewerSuggestionKind describes how strongly a reviewer is suggested for a pull request.
type ReviewerSuggestionKind string

func (ReviewerSuggestionKind) Enum() []any { return toInterfaceSlice(ReviewerSuggestionKinds) }
func (t ReviewerSuggestionKind) Sanitize() (ReviewerSuggestionKind, bool) {
	return Sanitize(t, GetAllReviewerSuggestionKinds)
}
func GetAllReviewerSuggestionKinds() ([]ReviewerSuggestionKind, ReviewerSuggestionKind) {
	return ReviewerSuggestionKinds, ReviewerSuggestionKindOptional
}

const (
	ReviewerSuggestionKindRequired  ReviewerSuggestionKind = "required"
	ReviewerSuggestionKindOptional  ReviewerSuggestionKind = "optional"
	ReviewerSuggestionKindNotNeeded ReviewerSuggestionKind = "not_needed"
)

var ReviewerSuggestionKinds = sortEnum([]ReviewerSuggestionKind{
	ReviewerSuggestionKindRequired,
	ReviewerSuggestionKindOptional,
	ReviewerSuggestionKindNotNeeded,
})
