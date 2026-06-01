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

// PullReqActivityMetadata contains metadata related to pull request activity.
type PullReqActivityMetadata struct {
	Suggestions *PullReqActivitySuggestionsMetadata `json:"suggestions,omitempty"`
	Mentions    *PullReqActivityMentionsMetadata    `json:"mentions,omitempty"`
	Reactions   *PullReqActivityReactionsMetadata   `json:"reactions,omitempty"`
}

func (m *PullReqActivityMetadata) IsEmpty() bool {
	return m == nil || (m.Suggestions == nil && m.Mentions == nil && m.Reactions.IsEmpty())
}

// PullReqActivityReactionsMetadata stores emoji reactions as a map of emoji name to reactor principal IDs.
type PullReqActivityReactionsMetadata struct {
	Counts map[string][]int64 `json:"counts,omitempty"`
}

func (m *PullReqActivityReactionsMetadata) IsEmpty() bool {
	return m == nil || len(m.Counts) == 0
}

type PullReqActivityMetadataUpdate interface {
	apply(m *PullReqActivityMetadata)
}

type pullReqActivityMetadataUpdateFunc func(m *PullReqActivityMetadata)

func (f pullReqActivityMetadataUpdateFunc) apply(m *PullReqActivityMetadata) {
	f(m)
}

func WithPullReqActivityMetadataUpdate(f func(m *PullReqActivityMetadata)) PullReqActivityMetadataUpdate {
	return pullReqActivityMetadataUpdateFunc(f)
}

// PullReqActivitySuggestionsMetadata contains metadata for code comment suggestions.
type PullReqActivitySuggestionsMetadata struct {
	CheckSums        []string `json:"check_sums,omitempty"`
	AppliedCheckSum  string   `json:"applied_check_sum,omitempty"`
	AppliedCommitSHA string   `json:"applied_commit_sha,omitempty"`
}

func (m *PullReqActivitySuggestionsMetadata) IsEmpty() bool {
	return len(m.CheckSums) == 0 && m.AppliedCheckSum == "" && m.AppliedCommitSHA == ""
}

func WithPullReqActivitySuggestionsMetadataUpdate(
	f func(m *PullReqActivitySuggestionsMetadata),
) PullReqActivityMetadataUpdate {
	return pullReqActivityMetadataUpdateFunc(func(m *PullReqActivityMetadata) {
		if m.Suggestions == nil {
			m.Suggestions = &PullReqActivitySuggestionsMetadata{}
		}

		f(m.Suggestions)

		if m.Suggestions.IsEmpty() {
			m.Suggestions = nil
		}
	})
}

// PullReqActivityMentionsMetadata contains metadata for code comment mentions.
type PullReqActivityMentionsMetadata struct {
	IDs          []int64 `json:"ids,omitempty"`
	UserGroupIDs []int64 `json:"user_group_ids,omitempty"`
}

func (m *PullReqActivityMentionsMetadata) IsEmpty() bool {
	return len(m.IDs) == 0 && len(m.UserGroupIDs) == 0
}

func WithPullReqActivityMentionsMetadataUpdate(
	f func(m *PullReqActivityMentionsMetadata),
) PullReqActivityMetadataUpdate {
	return pullReqActivityMetadataUpdateFunc(func(m *PullReqActivityMetadata) {
		if m.Mentions == nil {
			m.Mentions = &PullReqActivityMentionsMetadata{}
		}

		f(m.Mentions)

		if m.Mentions.IsEmpty() {
			m.Mentions = nil
		}
	})
}
