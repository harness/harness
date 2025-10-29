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

func (p *RepoTarget) Matches(repoID int64, repoIdentifier string) bool {
	// Note: exclusion always "wins" — if a repo is excluded, nothing can override it

	if slices.Contains(p.Exclude.IDs, repoID) {
		return false
	}

	for _, pattern := range p.Exclude.Patterns {
		if patternMatches(pattern, repoIdentifier) {
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
		if patternMatches(include, repoIdentifier) {
			return true
		}
	}

	return false
}

func matchesRepo(rawPattern json.RawMessage, repoID int64, repoIdentifier string) (bool, error) {
	repoTarget := RepoTarget{}
	if err := json.Unmarshal(rawPattern, &repoTarget); err != nil {
		return false, fmt.Errorf("failed to parse repo target: %w", err)
	}

	return repoTarget.Matches(repoID, repoIdentifier), nil
}
