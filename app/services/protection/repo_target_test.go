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
		repoPath  string
		spacePath string
		wantMatch bool
	}{
		{
			name: "exclude multiple ids, id in list",
			target: RepoTarget{
				Exclude: RepoTargetFilter{IDs: []int64{1, 2, 3, 4, 5}},
			},
			repoID:    3,
			repoPath:  "space/whatever",
			spacePath: "space",
			wantMatch: false,
		},
		{
			name: "exclude multiple ids, id not in list",
			target: RepoTarget{
				Exclude: RepoTargetFilter{IDs: []int64{1, 2, 3, 4, 5}},
			},
			repoID:    9,
			repoPath:  "space/whatever",
			spacePath: "space",
			wantMatch: true,
		},
		{
			name: "include multiple ids, id in list",
			target: RepoTarget{
				Include: RepoTargetFilter{IDs: []int64{7, 8, 9, 10, 11}},
			},
			repoID:    10,
			repoPath:  "space/some-repo",
			spacePath: "space",
			wantMatch: true,
		},
		{
			name: "include multiple ids, id not in list",
			target: RepoTarget{
				Include: RepoTargetFilter{IDs: []int64{7, 8, 9, 10, 11}},
			},
			repoID:    6,
			repoPath:  "space/some-repo",
			spacePath: "space",
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
			repoPath:  "space/test-nothing",
			spacePath: "space",
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
			repoPath:  "space/test-nothing",
			spacePath: "space",
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
			repoPath:  "space/boring-repo",
			spacePath: "space",
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
			repoPath:  "space/cool-repo",
			spacePath: "space",
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
			repoPath:  "space/boring-repo",
			spacePath: "space",
			wantMatch: false,
		},
		{
			name: "exclude and include multiple ids, exclude wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{IDs: []int64{1, 2, 3}},
				Include: RepoTargetFilter{IDs: []int64{1, 2, 3, 4}},
			},
			repoID:    2,
			repoPath:  "space/match-any",
			spacePath: "space",
			wantMatch: false,
		},
		{
			name: "exclude and include multiple ids, include wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{IDs: []int64{5, 6}},
				Include: RepoTargetFilter{IDs: []int64{7, 8, 9}},
			},
			repoID:    8,
			repoPath:  "space/match-any",
			spacePath: "space",
			wantMatch: true,
		},
		{
			name: "exclude and include multiple patterns, exclude wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"foo-*", "bar-*"}},
				Include: RepoTargetFilter{Patterns: []string{"foo-*", "baz-*"}},
			},
			repoID:    100,
			repoPath:  "space/bar-test",
			spacePath: "space",
			wantMatch: false,
		},
		{
			name: "exclude and include multiple patterns, include wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"foo-*", "bar-*"}},
				Include: RepoTargetFilter{Patterns: []string{"baz-*", "zoo-*"}},
			},
			repoID:    100,
			repoPath:  "space/zoo-special",
			spacePath: "space",
			wantMatch: true,
		},
		{
			name: "exclude and include patterns overlap, exclude wins",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"common-*"}},
				Include: RepoTargetFilter{Patterns: []string{"common-*", "rare-*"}},
			},
			repoID:    100,
			repoPath:  "space/common-42",
			spacePath: "space",
			wantMatch: false,
		},
		{
			name: "exclude and include patterns, include wins (not excluded)",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"foo-*"}},
				Include: RepoTargetFilter{Patterns: []string{"bar-*", "baz-*"}},
			},
			repoID:    100,
			repoPath:  "space/baz-42",
			spacePath: "space",
			wantMatch: true,
		},
		{
			name: "exclude and include patterns, neither matches",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"foo-*"}},
				Include: RepoTargetFilter{Patterns: []string{"bar-*", "baz-*"}},
			},
			repoID:    100,
			repoPath:  "space/other-42",
			spacePath: "space",
			wantMatch: false,
		},
		// relative path matching: rule defined at parent space, repo in child space
		{
			name: "space-level rule: pattern matches relative path with sub-space",
			target: RepoTarget{
				Include: RepoTargetFilter{Patterns: []string{"project/*"}},
			},
			repoID:    200,
			repoPath:  "org/project/my-repo",
			spacePath: "org",
			wantMatch: true,
		},
		{
			name: "space-level rule: pattern does not match different sub-space",
			target: RepoTarget{
				Include: RepoTargetFilter{Patterns: []string{"project/*"}},
			},
			repoID:    201,
			repoPath:  "org/other/my-repo",
			spacePath: "org",
			wantMatch: false,
		},
		{
			name: "space-level rule: exclusion uses relative path",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"project/my-repo"}},
			},
			repoID:    202,
			repoPath:  "org/project/my-repo",
			spacePath: "org",
			wantMatch: false,
		},
		{
			name: "space-level rule: exclusion does not affect sibling repo",
			target: RepoTarget{
				Exclude: RepoTargetFilter{Patterns: []string{"project/my-repo"}},
			},
			repoID:    203,
			repoPath:  "org/project/other-repo",
			spacePath: "org",
			wantMatch: true,
		},
		// deeply nested: rule at grandparent, repo three levels deep
		{
			name: "nested space: grandparent rule matches deep relative path",
			target: RepoTarget{
				Include: RepoTargetFilter{Patterns: []string{"team/project/*"}},
			},
			repoID:    300,
			repoPath:  "org/team/project/my-repo",
			spacePath: "org",
			wantMatch: true,
		},
		{
			name: "nested space: grandparent rule does not match wrong branch",
			target: RepoTarget{
				Include: RepoTargetFilter{Patterns: []string{"team/project/*"}},
			},
			repoID:    301,
			repoPath:  "org/team/other/my-repo",
			spacePath: "org",
			wantMatch: false,
		},
		// repo-level rule (spacePath empty): uses bare identifier (last segment)
		{
			name: "repo-level rule (spacePath empty): pattern matches bare identifier",
			target: RepoTarget{
				Include: RepoTargetFilter{Patterns: []string{"cool-*"}},
			},
			repoID:    50,
			repoPath:  "space/sub/cool-repo",
			spacePath: "",
			wantMatch: true,
		},
		{
			name: "repo-level rule (spacePath empty): pattern does not match sibling segment",
			target: RepoTarget{
				Include: RepoTargetFilter{Patterns: []string{"cool-*"}},
			},
			repoID:    51,
			repoPath:  "space/sub/boring-repo",
			spacePath: "",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.target.Matches(tt.repoID, tt.repoPath, tt.spacePath)
			if got != tt.wantMatch {
				t.Errorf("Matches(%d, %q, %q) = %v; want %v", tt.repoID, tt.repoPath, tt.spacePath, got, tt.wantMatch)
			}
		})
	}
}
