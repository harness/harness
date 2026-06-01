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

import (
	"testing"
)

func TestPullReqActivityReactionsMetadataIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		m    *PullReqActivityReactionsMetadata
		want bool
	}{
		{"nil", nil, true},
		{"empty counts", &PullReqActivityReactionsMetadata{}, true},
		{"empty map", &PullReqActivityReactionsMetadata{Counts: map[string][]int64{}}, true},
		{"one reaction", &PullReqActivityReactionsMetadata{Counts: map[string][]int64{"plusone": {1}}}, false},
		{
			"multiple emojis",
			&PullReqActivityReactionsMetadata{Counts: map[string][]int64{"plusone": {1, 2}, "smile": {3}}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPullReqActivityMetadataIsEmpty(t *testing.T) {
	reactions := &PullReqActivityReactionsMetadata{Counts: map[string][]int64{"plusone": {1}}}
	mentions := &PullReqActivityMentionsMetadata{IDs: []int64{1}}
	suggestions := &PullReqActivitySuggestionsMetadata{CheckSums: []string{"abc"}}

	tests := []struct {
		name string
		m    *PullReqActivityMetadata
		want bool
	}{
		{"nil", nil, true},
		{"all nil fields", &PullReqActivityMetadata{}, true},
		{"only reactions", &PullReqActivityMetadata{Reactions: reactions}, false},
		{"only mentions", &PullReqActivityMetadata{Mentions: mentions}, false},
		{"only suggestions", &PullReqActivityMetadata{Suggestions: suggestions}, false},
		{"all set", &PullReqActivityMetadata{Suggestions: suggestions, Mentions: mentions, Reactions: reactions}, false},
		{"empty reactions", &PullReqActivityMetadata{Reactions: &PullReqActivityReactionsMetadata{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
