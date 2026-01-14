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

package protection

import "testing"

func TestRepoTarget_Matches(t *testing.T) {
	tests := []struct {
		name      string
		target    RepoTarget
		repoID    int64
		repoUID   string
		wantMatch bool
	}{
		{
			name: "exclude multiple ids, id in list",
			target: RepoTarget{
				Exclude: RepoTargetFilter{IDs: []int64{1, 2, 3, 4, 5}},
			},
			repoID:    3,
			repoUID:   "whatever",
			wantMatch: false,
		},
		{
			name: "exclude multiple ids, id not in list",
			target: RepoTarget{
				Exclude: RepoTargetFilter{IDs: []int64{1, 2, 3, 4, 5}},
			},
			repoID:    9,
			repoUID:   "whatever",
			wantMatch: true,
		},
		{
			name: "include multiple ids, id in list",
			target: RepoTarget{
				Include: RepoTargetFilter{IDs: []int64{7, 8, 9, 10, 11}},
			},
			repoID:    10,
			repoUID:   "some-repo",
			wantMatch: true,
		},
		{
			name: "include multiple ids, id not in list",
			target: RepoTarget{
				Include: RepoTargetFilter{IDs: []int64{7, 8, 9, 10, 11}},
			},
			repoID:    6,
			repoUID:   "some-repo",
			wantMatch: false,
		},
		{
			name: "exclude ids and patterns, id match wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{
					IDs:      []int64{13, 14, 15},
					Patterns: []string{"test-*"},
				},
			},
			repoID:    14,
			repoUID:   "test-nothing",
			wantMatch: false,
		},
		{
			name: "exclude ids and patterns, pattern match wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{
					IDs:      []int64{13, 14, 15},
					Patterns: []string{"test-*"},
				},
			},
			repoID:    20,
			repoUID:   "test-nothing",
			wantMatch: false,
		},
		{
			name: "include multiple ids and patterns, id match wins",
			target: RepoTarget{
				Include: RepoTargetFilter{
					IDs:      []int64{21, 22, 23},
					Patterns: []string{"cool-*"},
				},
			},
			repoID:    21,
			repoUID:   "boring-repo",
			wantMatch: true,
		},
		{
			name: "include multiple ids and patterns, pattern match wins",
			target: RepoTarget{
				Include: RepoTargetFilter{
					IDs:      []int64{21, 22, 23},
					Patterns: []string{"cool-*"},
				},
			},
			repoID:    30,
			repoUID:   "cool-repo",
			wantMatch: true,
		},
		{
			name: "include multiple ids and patterns, match neither",
			target: RepoTarget{
				Include: RepoTargetFilter{
					IDs:      []int64{21, 22, 23},
					Patterns: []string{"cool-*"},
				},
			},
			repoID:    99,
			repoUID:   "boring-repo",
			wantMatch: false,
		},
		{
			name: "exclude and include multiple ids, exclude wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{IDs: []int64{1, 2, 3}},
				Include: RepoTargetFilter{IDs: []int64{1, 2, 3, 4}},
			},
			repoID:    2,
			repoUID:   "match-any",
			wantMatch: false,
		},
		{
			name: "exclude and include multiple ids, include wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{IDs: []int64{5, 6}},
				Include: RepoTargetFilter{IDs: []int64{7, 8, 9}},
			},
			repoID:    8,
			repoUID:   "match-any",
			wantMatch: true,
		},
		{
			name: "exclude and include multiple patterns, exclude wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"foo-*", "bar-*"}},
				Include: RepoTargetFilter{Patterns: []string{"foo-*", "baz-*"}},
			},
			repoID:    100,
			repoUID:   "bar-test",
			wantMatch: false,
		},
		{
			name: "exclude and include multiple patterns, include wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"foo-*", "bar-*"}},
				Include: RepoTargetFilter{Patterns: []string{"baz-*", "zoo-*"}},
			},
			repoID:    100,
			repoUID:   "zoo-special",
			wantMatch: true,
		},
		{
			name: "exclude and include patterns overlap, exclude wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"common-*"}},
				Include: RepoTargetFilter{Patterns: []string{"common-*", "rare-*"}},
			},
			repoID:    100,
			repoUID:   "common-42",
			wantMatch: false,
		},
		{
			name: "exclude and include patterns, include wins (not excluded)",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"foo-*"}},
				Include: RepoTargetFilter{Patterns: []string{"bar-*", "baz-*"}},
			},
			repoID:    100,
			repoUID:   "baz-42",
			wantMatch: true,
		},
		{
			name: "exclude and include patterns, neither matches",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"foo-*"}},
				Include: RepoTargetFilter{Patterns: []string{"bar-*", "baz-*"}},
			},
			repoID:    100,
			repoUID:   "other-42",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.target.Matches(tt.repoID, tt.repoUID)
			if got != tt.wantMatch {
				t.Errorf("Matches(%d, %q) = %v; want %v", tt.repoID, tt.repoUID, got, tt.wantMatch)
			}
		})
	}
}
