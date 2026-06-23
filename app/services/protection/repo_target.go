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

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type RepoTargetFilter struct {
	IDs      []int64  `json:"ids,omitempty"`
	Patterns []string `json:"patterns,omitempty"`
}

type RepoTarget struct {
	Include RepoTargetFilter `json:"include"`
	Exclude RepoTargetFilter `json:"exclude"`
}

func (p *RepoTarget) JSON() json.RawMessage {
	message, _ := ToJSON(p)
	return message
}

func (p *RepoTarget) Validate() error {
	if err := validateIDSlice(p.Include.IDs); err != nil {
		return err
	}

	for _, pattern := range p.Include.Patterns {
		if err := patternValidate(pattern); err != nil {
			return err
		}
	}

	for _, pattern := range p.Exclude.Patterns {
		if err := patternValidate(pattern); err != nil {
			return err
		}
	}

	if err := validateIDSlice(p.Exclude.IDs); err != nil {
		return err
	}

	return nil
}

// repoRelPath returns the repository path relative to the given space path.
// For a space-level rule at spacePath "org/project" and repoPath "org/project/my-repo",
// it returns "my-repo". For nested repos like "org/project/sub/repo" it returns "sub/repo".
// If spacePath is empty (repo-level rule), the last segment of repoPath is returned.
func repoRelPath(repoPath, spacePath string) string {
	if spacePath == "" {
		// repo-level rule: use the bare identifier (last path segment)
		if idx := strings.LastIndex(repoPath, "/"); idx >= 0 {
			return repoPath[idx+1:]
		}
		return repoPath
	}
	prefix := spacePath + "/"
	if strings.HasPrefix(repoPath, prefix) {
		return repoPath[len(prefix):]
	}
	// Shouldn't happen: a repo always lives under its space. Return the bare
	// identifier (last segment) rather than the full path so that if this ever
	// fires due to data inconsistency, patterns still have a sensible value to
	// match against instead of an unexpected full path.
	if idx := strings.LastIndex(repoPath, "/"); idx >= 0 {
		return repoPath[idx+1:]
	}
	return repoPath
}

func (p *RepoTarget) Matches(repoID int64, repoPath, spacePath string) bool {
	// Note: exclusion always "wins" — if a repo is excluded, nothing can override it

	if slices.Contains(p.Exclude.IDs, repoID) {
		return false
	}

	if slices.Contains(p.Include.IDs, repoID) {
		return true
	}

	relPath := repoRelPath(repoPath, spacePath)

	for _, pattern := range p.Exclude.Patterns {
		if patternMatches(pattern, relPath) {
			return false
		}
	}

	// If either includes are unspecified (empty), "match all"
	if len(p.Include.IDs) == 0 && len(p.Include.Patterns) == 0 {
		return true
	}

	// If includes are specified, a repo must match at least one
	if slices.Contains(p.Include.IDs, repoID) {
		return true
	}

	for _, include := range p.Include.Patterns {
		if patternMatches(include, relPath) {
			return true
		}
	}

	return false
}

func matchesRepo(rawPattern json.RawMessage, repoID int64, repoPath, spacePath string) (bool, error) {
	repoTarget := RepoTarget{}
	if err := json.Unmarshal(rawPattern, &repoTarget); err != nil {
		return false, fmt.Errorf("failed to parse repo target: %w", err)
	}

	return repoTarget.Matches(repoID, repoPath, spacePath), nil
}
