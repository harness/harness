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

type (
	SearchInput struct {
		Query string `json:"query"`

		// RepoPaths contains the paths of repositories to search in
		RepoPaths []string `json:"repo_paths"`

		// SpacePaths contains the paths of spaces to search in
		SpacePaths []string `json:"space_paths"`

		// MaxResultCount is the maximum number of results to return
		MaxResultCount int `json:"max_result_count"`

		// EnableRegex enables regex search on the query
		EnableRegex bool `json:"enable_regex"`

		// Search all the repos in a space and its subspaces recursively.
		// Valid only when spacePaths is set.
		Recursive bool `json:"recursive"`
	}

	SearchResult struct {
		FileMatches []FileMatch `json:"file_matches"`
		Stats       SearchStats `json:"stats"`
	}

	SearchStats struct {
		TotalFiles   int `json:"total_files"`
		TotalMatches int `json:"total_matches"`
	}

	FileMatch struct {
		FileName   string  `json:"file_name"`
		RepoID     int64   `json:"-"`
		RepoPath   string  `json:"repo_path"`
		RepoBranch string  `json:"repo_branch"`
		Language   string  `json:"language"`
		Matches    []Match `json:"matches"`
	}

	// Match holds the per line data.
	Match struct {
		// LineNum is the line number of the match
		LineNum int `json:"line_num"`

		// Fragments holds the matched fragments within the line
		Fragments []Fragment `json:"fragments"`

		// Before holds the content from the line immediately preceding the line where the match was found
		Before string `json:"before"`

		// After holds the content from the line immediately following the line where the match was found
		After string `json:"after"`
	}

	// Fragment holds data of a single contiguous match within a line.
	Fragment struct {
		Pre   string `json:"pre"`   // the string before the match within the line
		Match string `json:"match"` // the matched string
		Post  string `json:"post"`  // the string after the match within the line
	}
)
